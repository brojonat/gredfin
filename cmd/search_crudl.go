package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/brojonat/gredfin/server"
	"github.com/jackc/pgx/v5/pgtype"
)

func AddSeachQuery(ctx context.Context, l *slog.Logger, endpoint, authToken, q string) error {
	b, err := json.Marshal(pgtype.Text{String: q, Valid: true})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/search", endpoint),
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
