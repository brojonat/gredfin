package worker

import (
	"context"
	"log/slog"
	"time"
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
	// TODO: set this in the past to trigger immediate run? The first run doesn't
	// happen until after a delay.
	lastRun := time.Now()
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
