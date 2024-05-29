package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server"
	"github.com/brojonat/gredfin/server/dbgen"
)

// Default implementation of a Property scrape worker.
func MakePropertyWorkerFunc(
	endpoint string,
	authToken string,
	grc redfin.Client,
) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, l *slog.Logger) {
		l.Info("running property scrape worker")
		p, err := claimProperty(endpoint, server.GetDefaultServerHeaders(authToken))
		if err != nil {
			l.Error("error getting property", "error", err.Error())
			return
		}
		l.Info("got property", "url", p.URL.String)

		// pull from redfin and upload to cloud
	}
	return f
}

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
