package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/brojonat/gredfin/client"
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

// no-op worker function for testing
func NoopWorkerFunc(ctx context.Context, logger *slog.Logger) {
	logger.Info("look at me, i'm doing search work")
	time.Sleep(5 * time.Second)
	logger.Info("done search work")
}

// Default implementation of a Search scrape worker.
func MakeSearchWorkerFunc(grc client.Client, s3c *s3.Client) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, logger *slog.Logger) {
		// logger.Info("look at me, I'm a search scraper with dependencies")
		// values, err := s3c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String("gredfin"), Prefix: aws.String("idk")})
		// if err != nil {
		// 	logger.Error("error fetching data", "error", err.Error())
		// 	return
		// }
		// data, err := json.Marshal(values.Contents)
		// if err != nil {
		// 	logger.Error("error serializing bucket data", "error", err.Error())
		// 	return
		// }
		// logger.Info("got some values", "values", string(data))
		logger.Info("I'm a search worker!")
	}
	return f
}

// Default implementation of a Property scrape worker.
func MakePropertyWorkerFunc(grc client.Client, s3c *s3.Client) func(context.Context, *slog.Logger) {
	f := func(ctx context.Context, logger *slog.Logger) {
		// logger.Info("look at me, I'm a property scraper with dependencies")
		// values, err := s3c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String("gredfin"), Prefix: aws.String("idk")})
		// if err != nil {
		// 	logger.Error("error fetching data", "error", err.Error())
		// 	return
		// }
		// data, err := json.Marshal(values.Contents)
		// if err != nil {
		// 	logger.Error("error serializing bucket data", "error", err.Error())
		// 	return
		// }
		// logger.Info("got some values", "values", string(data))
		logger.Info("I'm a property worker!")
	}
	return f
}
