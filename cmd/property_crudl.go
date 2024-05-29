package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/server/dbgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func AddPropertyQuery(ctx context.Context, l *slog.Logger, endpoint, authToken, property_id, listing_id, url string) error {
	pid, err := strconv.Atoi(property_id)
	if err != nil {
		return err
	}
	lid, err := strconv.Atoi(listing_id)
	if err != nil {
		return err
	}
	b, err := json.Marshal(dbgen.CreatePropertyParams{
		PropertyID: int32(pid),
		ListingID:  int32(lid),
		URL:        pgtype.Text{String: url, Valid: true},
	})
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
	req.Header = server.GetDefaultServerHeaders(authToken)
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
