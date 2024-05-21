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

func uploadSearchResults(endpoint string, h http.Header, l *slog.Logger, b []byte) {
	// get the CSV of properties for this search and dump it into a csv
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
		addr := fmt.Sprintf("%s, %s, %s, %s", row[3], row[4], row[5], row[6])
		err := createSearch(endpoint, h, pgtype.Text{String: addr})
		if err != nil {
			l.Error(
				"error creating property",
				"error", err.Error(),
				"address", addr,
			)
			continue
		}
	}
}

func createSearch(endpoint string, h http.Header, addr pgtype.Text) error {
	b, err := json.Marshal(addr)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/search-query", endpoint),
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
