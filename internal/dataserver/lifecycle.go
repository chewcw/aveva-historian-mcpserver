package dataserver

import (
	"context"
	"log/slog"
	"time"
)

func RunGC(ctx context.Context, store *Store, interval time.Duration, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("source", "dataserver.GC")
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n := store.Sweep(time.Now())
				if n > 0 {
					logger.Debug("gc sweep complete", "removed", n)
				}
			}
		}
	}()
}
