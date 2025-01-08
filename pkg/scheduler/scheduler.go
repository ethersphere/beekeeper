package scheduler

import (
	"context"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"resenje.org/x/shutdown"
)

type PeriodicExecutor struct {
	ticker   *time.Ticker
	interval time.Duration
	log      logging.Logger
	shutdown *shutdown.Graceful
}

func NewPeriodicExecutor(interval time.Duration, log logging.Logger) *PeriodicExecutor {
	return &PeriodicExecutor{
		ticker:   time.NewTicker(interval),
		interval: interval,
		log:      log,
		shutdown: shutdown.NewGraceful(),
	}
}

func (pe *PeriodicExecutor) Start(ctx context.Context, task func(ctx context.Context) error) {
	pe.shutdown.Add(1)
	go func() {
		defer pe.shutdown.Done()
		ctx = pe.shutdown.Context(ctx)

		if err := task(ctx); err != nil {
			pe.log.Errorf("Task execution failed: %v", err)
		}

		for {
			select {
			case <-pe.ticker.C:
				pe.log.Tracef("Executing task after %s interval", pe.interval)
				if err := task(ctx); err != nil {
					pe.log.Errorf("Task execution failed: %v", err)
				}
			case <-pe.shutdown.Quit():
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (pe *PeriodicExecutor) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pe.ticker.Stop()
	return pe.shutdown.Shutdown(ctx)
}
