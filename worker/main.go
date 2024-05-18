package worker

import (
	"context"
	"log/slog"
	"time"
)

func RunSearchWorker(ctx context.Context, logger *slog.Logger, interval, olderThan time.Duration) error {
	select {}
	return nil
}

func RunPropertyWorker(ctx context.Context, logger *slog.Logger, interval, olderThan time.Duration) error {
	select {}
	return nil
}
