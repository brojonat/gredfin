package worker

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/brojonat/gredfin/redfin"
)

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
func MakeSearchWorkerFunc(endpoint string, authToken string, grc redfin.Client, s3c *s3.Client) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, logger *slog.Logger) {
		// claim the search query
		logger.Info("running search scrape worker")
		s, err := claimSearch(endpoint, getDefaultHeaders(authToken))
		if err != nil {
			logger.Error("error getting search", "error", err.Error())
			return
		}
		logger.Info("got query", "query", s.Query)

		// run query (b is a byte slice representing CSV of search results)
		b := []byte{}

		// upload results
		uploadSearchResults(endpoint, getDefaultHeaders(authToken), logger, b)
	}
	return f
}

// Default implementation of a Property scrape worker.
func MakePropertyWorkerFunc(endpoint string, authToken string, grc redfin.Client, s3c *s3.Client) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, logger *slog.Logger) {
		logger.Info("running property scrape worker")
		p, err := claimProperty(endpoint, getDefaultHeaders(authToken))
		if err != nil {
			logger.Error("error getting property", "error", err.Error())
			return
		}
		logger.Info("got property", "address", p.Address)
	}
	return f
}

func getDefaultHeaders(authToken string) http.Header {
	h := http.Header{}
	h.Add("Authorization", authToken)
	h.Add("Content-Type", "application/json")
	return h
}
