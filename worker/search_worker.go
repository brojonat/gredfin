package worker

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
)

// POST a request to claim an item from the search queue.
func claimSearch(endpoint string, h http.Header) (*dbgen.Search, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/search-query/claim-next", endpoint),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%s (likely no available search queries to run)", res.Status)
		}
		return nil, fmt.Errorf(res.Status)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var s dbgen.Search
	err = json.Unmarshal(b, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func markSearchStatus(endpoint string, h http.Header, s *dbgen.Search, status string) error {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/search-query/set-status", endpoint),
		nil,
	)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("search_id", strconv.Itoa(int(s.SearchID)))
	q.Add("status", status)
	req.URL.RawQuery = q.Encode()
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(res.Status)
	}
	return nil
}

func getURLSFromQuery(
	l *slog.Logger,
	grc redfin.Client,
	query string,
	searchParams map[string]string,
	giscsvParams map[string]string,
) ([]string, error) {
	// first run a vanilla search using the supplied query (should be a zip code)
	b, err := grc.Search(query, searchParams)
	if err != nil {
		return nil, fmt.Errorf("error running search query: %w", err)
	}
	var res redfin.RedfinResponse
	if err = json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("error serializing initial_info response: %w", err)
	}

	// extract the Redfin region for this zipcode
	var p redfin.SearchPayload
	if err = json.Unmarshal(res.Payload, &p); err != nil {
		return nil, fmt.Errorf("error serializing search payload: %w", err)
	}
	if len(p.Sections) < 1 {
		l.Error("logging bad search payload for reference", "payload", string(b))
		return nil, fmt.Errorf("error extracting region from search: no Sections")
	}
	if len(p.Sections[0].Rows) < 1 {
		l.Error("logging bad search payload for reference", "payload", string(b))
		return nil, fmt.Errorf("error extracting region from search: no Rows")
	}
	regionParts := strings.Split(p.Sections[0].Rows[0].ID, "_")
	if len(regionParts) != 2 {
		l.Error("logging bad search payload for reference", "payload", string(b))
		return nil, fmt.Errorf("unexpected region format: %s", p.Sections[0].Rows[0].ID)
	}

	// giscsvParams["region_type"] = regionParts[0]
	giscsvParams["region_id"] = regionParts[1]
	b, err = grc.GISCSV(giscsvParams)

	if err != nil {
		return nil, fmt.Errorf("error getting csv: %w", err)
	}

	csvr := csv.NewReader(bytes.NewReader(b))
	csvr.FieldsPerRecord = -1 // Allow variable number of fields
	rows, err := csvr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading csv bytes: %w", err)
	}

	headers := rows[0]
	_ = rows[1] // In accordance with local MLS rules, some MLS listings are not included in the download
	data := rows[2:]
	var urlIndex int
	for idx, h := range headers {
		if strings.HasPrefix(h, "URL") {
			urlIndex = idx
			break
		}
	}

	urls := []string{}
	for _, row := range data {
		urls = append(urls, row[urlIndex])
	}
	return urls, nil
}

func addPropertyFromURL(
	endpoint string,
	h http.Header,
	grc redfin.Client,
	url string,
	delay time.Duration,
) error {
	bii, err := grc.InitialInfo(
		strings.TrimPrefix(url, "https://www.redfin.com"),
		map[string]string{},
	)
	// sleep to avoid smashing redfin's api
	time.Sleep(delay)
	if err != nil {
		return fmt.Errorf("error getting initial_info: %w", err)
	}
	var res redfin.RedfinResponse
	if err = json.Unmarshal(bii, &res); err != nil {
		return fmt.Errorf("error serializing initial_info response: %w", err)
	}
	var iip redfin.InitialInfoPayload
	if err = json.Unmarshal(res.Payload, &iip); err != nil {
		return fmt.Errorf("error serializing initial_info payload: %w", err)
	}

	p := &dbgen.CreatePropertyParams{
		PropertyID: int32(iip.PropertyID),
		ListingID:  int32(iip.ListingID),
		URL:        pgtype.Text{String: url, Valid: true},
	}
	if err = createProperty(endpoint, h, p); err != nil {
		return fmt.Errorf(
			"error creating property (property_id: %d, listing_id: %d, url: %s): %w",
			iip.PropertyID, iip.ListingID, url, err,
		)
	}
	return nil
}
