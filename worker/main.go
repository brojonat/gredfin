package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/brojonat/gredfin/redfin"
	"github.com/brojonat/gredfin/server"
)

func AddSeachQuery(ctx context.Context, l *slog.Logger, endpoint, authToken, q string) error {
	b, err := json.Marshal(pgtype.Text{String: q, Valid: true})
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
	req.Header = getDefaultServerHeaders(authToken)
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

// RunWorkerFunc is a general purpose entry point for running cancelable
// periodic worker functions on some interval. Callers simply supply an interval
// and their worker function.
func RunWorkerFunc(
	ctx context.Context,
	logger *slog.Logger,
	interval time.Duration,
	f func(context.Context, *slog.Logger),
) error {
	lastRun := time.Now() // TODO: set this in the past to trigger immediate run?
	for {
		delay := time.NewTimer(lastRun.Truncate(interval).Add(interval).Sub(lastRun))
		select {
		case <-delay.C:
			f(ctx, logger)
		case <-ctx.Done():
			logger.Info("search worker context cancelled, return context err")
			if !delay.Stop() {
				logger.Info("flushing work timer thingy")
				<-delay.C
			}
			return ctx.Err()
		}
		lastRun = time.Now()
	}
}

// Default implementation of a Search scrape worker.
func MakeSearchWorkerFunc(
	endpoint string,
	authToken string,
	grc redfin.Client,
	s3c *s3.Client,
) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, l *slog.Logger) {
		// claim the search query
		l.Info("running search scrape worker loop")
		s, err := claimSearch(endpoint, getDefaultServerHeaders(authToken))
		if err != nil {
			l.Error("error getting search, exiting", "error", err.Error())
			return
		}
		l.Info("claimed query", "query", s.Query.String)

		// run the query and get a list of Redfin URLs
		urls, err := getURLSFromQuery(
			l,
			grc,
			s.Query.String,
			getDefaultSearchParams(),
			getDefaultGISCSVParams(),
		)
		if err != nil {
			l.Error(err.Error())
			return
		}

		// for each URL, upload the property listing to the DB
		h := getDefaultServerHeaders(authToken)
		for _, u := range urls {
			if err := addPropertyFromURL(endpoint, h, l, grc, u); err != nil {
				l.Error(err.Error())
			}
		}

		err = markSearchStatus(endpoint, getDefaultServerHeaders(authToken), s, server.ScrapeStatusGood)
		if err != nil {
			l.Error(err.Error())
			return
		}
	}
	return f
}

// Default implementation of a Property scrape worker.
func MakePropertyWorkerFunc(
	endpoint string,
	authToken string,
	grc redfin.Client,
	s3c *s3.Client,
) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, l *slog.Logger) {
		l.Info("running property scrape worker")
		p, err := claimProperty(endpoint, getDefaultServerHeaders(authToken))
		if err != nil {
			l.Error("error getting property", "error", err.Error())
			return
		}
		l.Info("got property", "url", p.URL.String)
		// pull from redfin and upload to cloud
	}
	return f
}
