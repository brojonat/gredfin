package worker

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func claimProperty(endpoint string, headers http.Header) (*dbgen.Property, error) {
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property-query/claim-next", endpoint),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header = headers
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(res.Status)
	}

	var p dbgen.Property
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func createProperty(endpoint string, h http.Header, c *dbgen.CreatePropertyParams) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/property-query", endpoint),
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return fmt.Errorf(res.Status)
	}
	return nil
}

func uploadSearchResults(endpoint string, h http.Header, l *slog.Logger, b []byte) {
	// get the CSV of properties from a search search and dump it into a csv
	file, err := os.CreateTemp("", "search")
	if err != nil {
		l.Error("error opening temp file", "error", err.Error())
		return
	}
	defer os.Remove(file.Name())
	file.Write(b)

	// read the csv
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	data, err := reader.ReadAll()
	if err != nil {
		l.Error("error reading CSV", "error", err.Error())
		return
	}

	// upload each property
	for _, row := range data {
		// the row only has an address, no property/listing ids, so we need to
		// run a search NOW to get that info. The good thing is we're working fro
		// a property so we can go after the exact match. FIXME: there's still some
		// work to implement here.
		propertyID := "123"
		listingID := "456"
		addr := row[5]
		cpp := &dbgen.CreatePropertyParams{
			PropertyID: propertyID,
			ListingID:  listingID,
			Address:    pgtype.Text{String: addr},
		}
		err := createProperty(endpoint, h, cpp)
		if err != nil {
			l.Error(
				"error creating property",
				"error", err.Error(),
				"propertyID", propertyID,
				"listingID", listingID,
				"address", addr,
			)
			continue
		}
	}
}
