package core

import (
	"context"
	"log/slog"
	"time"
)

func StartTimeWatchDog(ctx context.Context, logger *slog.Logger) <-chan struct{} {
	logger.Info("starting watchdog")
	ch := make(chan struct{}, 1)
	go func() {
		lastUnix := time.Now().Unix()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				nowUnix := time.Now().Unix()
				diff := nowUnix - lastUnix
				if diff > 300 {
					logger.Warn("system time jump detected(wake up from sleep?)",
						slog.Int64("jump_seconds", diff),
					)
					ch <- struct{}{}
					logger.Debug("signal sent, watchdog exiting...")
					return
				} else {
					lastUnix = nowUnix
				}
			}
		}
	}()
	return ch
}
