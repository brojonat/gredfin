package worker

import (
	"bytes"
	"context"
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
	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/server/db/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/twpayne/go-geos"
	"github.com/twpayne/go-geos/geometry"
)

// Default implementation of a Search scrape worker. The worker pulls a search
// query from the service and runs the query against the Redfin API. A list of
// URLs is extracted from the result and for each URL, a series of queries is
// performed against the RefinAPI
func MakeSearchWorkerFunc(
	endpoint string,
	authToken string,
	grc redfin.Client,
	pqd time.Duration,
) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, l *slog.Logger) {
		// claim the search query
		l.Info("running search scrape worker loop")
		s, err := claimSearch(endpoint, server.GetDefaultServerHeaders(authToken))
		if err != nil {
			l.Error("error getting search, exiting", "error", err.Error())
			return
		}
		l.Info("claimed query", "query", s.Query.String)

		// run the query and get a list of Redfin URLs
		urls, err := GetURLSFromQuery(
			l,
			grc,
			s.Query.String,
			GetDefaultSearchParams(),
			GetDefaultGISCSVParams(),
		)
		if err != nil {
			l.Error(err.Error())
			return
		}

		// for each URL, upload the property listing to the DB
		h := server.GetDefaultServerHeaders(authToken)
		nerr := 0
		nsuccess := len(urls)
		for _, u := range urls {
			if err := addPropertyFromURL(endpoint, h, grc, u, pqd); err != nil {
				l.Error(err.Error())
				nerr += 1
				nsuccess -= 1
			}
		}
		l.Info("search results uploaded", "error", nerr, "success", nsuccess)

		// If any properties are uploaded successfully, we consider that a
		// "good" scrape since there may be problematic properties returned that
		// we don't expect to be able to parse. A "bad" scrape is one that had
		// URLs returned and didn't successfully upload any properties to the
		// server. This may result in some scrapes getting marked bad when in
		// reality, by chance, they happen to not have any parseable properties,
		// but it's good to identify those searches anyway.
		status := server.ScrapeStatusGood
		if len(urls) > 0 && nsuccess == 0 {
			status = server.ScrapeStatusBad
		}
		if err = markSearchStatus(endpoint, server.GetDefaultServerHeaders(authToken), s, status, nsuccess, nerr); err != nil {
			l.Error(err.Error())
			return
		}
	}
	return f
}

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

func markSearchStatus(endpoint string, h http.Header, s *dbgen.Search, status string, success_count, error_count int) error {
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
	q.Add("success_count", strconv.Itoa(success_count))
	q.Add("error_count", strconv.Itoa(error_count))
	req.URL.RawQuery = q.Encode()
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var data server.DefaultJSONResponse
	if err = json.Unmarshal(b, &data); err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s (%s)", res.Status, data.Error)
	}
	return nil
}

func GetURLSFromQuery(
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
	giscsvParams["region_id"] = regionParts[1]
	b, err = grc.GISCSV(giscsvParams)
	if err != nil {
		return nil, fmt.Errorf("error getting csv: %w", err)
	}

	csvr := csv.NewReader(bytes.NewReader(b))
	csvr.FieldsPerRecord = -1 // Allow variable number of fields
	rows, err := csvr.ReadAll()
	if err != nil {
		l.Debug(fmt.Sprintf("%s", giscsvParams))
		l.Error(string(b))
		return nil, fmt.Errorf("error reading csv bytes: %w", err)
	}
	if len(rows) <= 2 {
		l.Debug("no rows for query", "query", query, "region", p.Sections[0].Rows[0].ID, "region_type", giscsvParams["region_type"])
		return []string{}, nil
	}

	headers := rows[0]
	// rows[1] should be: "In accordance with local MLS rules, some MLS listings are not included in the download"
	if rows[1][0] != "In accordance with local MLS rules, some MLS listings are not included in the download" {
		l.Error("unexpected rows format, missing MLS caveat line", "rows[1]", rows[1])
	}
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
	ts_start := time.Now()
	b, err := grc.InitialInfo(
		strings.TrimPrefix(url, "https://www.redfin.com"),
		map[string]string{},
	)
	if err != nil {
		return fmt.Errorf("error getting initial_info: %w", err)
	}
	var res redfin.RedfinResponse
	if err = json.Unmarshal(b, &res); err != nil {
		return fmt.Errorf("error parsing initial_info response: %w", err)
	}
	var jmesdata interface{}
	if err := json.Unmarshal(res.Payload, &jmesdata); err != nil {
		return fmt.Errorf("error parsing initial_info data: %w", err)
	}

	// parse property_id
	property_id, err := jmesParseInitialInfoParams("property_id", jmesdata)
	if err != nil {
		return fmt.Errorf("error searching for property_id: %w", err)
	}
	if property_id == nil {
		return fmt.Errorf("null result extracting property_id")
	}

	// parse listing_id
	listing_id, err := jmesParseInitialInfoParams("listing_id", jmesdata)
	if err != nil {
		return fmt.Errorf("error searching for listing_id: %w", err)
	}
	if listing_id == nil {
		return fmt.Errorf("null result extracting listing_id")
	}
	pid, lid := int(property_id.(float64)), int(listing_id.(float64))

	// parse lat/long
	lat, err := jmesParseInitialInfoParams("latitude", jmesdata)
	if err != nil {
		return fmt.Errorf("error searching for latitude: %w", err)
	}
	if lat == nil {
		return fmt.Errorf("null result extracting latitude")
	}
	long, err := jmesParseInitialInfoParams("longitude", jmesdata)
	if err != nil {
		return fmt.Errorf("error searching for longitude: %w", err)
	}
	if long == nil {
		return fmt.Errorf("null result extracting longitude")
	}
	latitude, longitude := lat.(float64), long.(float64)

	p := &dbgen.CreatePropertyParams{
		PropertyID: int32(pid),
		ListingID:  int32(lid),
		URL:        pgtype.Text{String: url, Valid: true},
		Location:   geometry.NewGeometry(geos.NewPoint([]float64{longitude, latitude}).SetSRID(4326)),
	}
	b, err = json.Marshal(p)
	if err != nil {
		return fmt.Errorf("error serializing create Property request: %w", err)
	}
	if err = createProperty(endpoint, h, b); err != nil {
		return fmt.Errorf(
			"error creating property (property_id: %d, listing_id: %d, url: %s): %w",
			pid, lid, url, err,
		)
	}

	// sleep to avoid smashing redfin's api
	ts_end := time.Now()
	time.Sleep(delay - ts_end.Sub(ts_start))
	return nil
}
