package scheduler

import (
	"context"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
)

type PeriodicExecutor struct {
	ticker   *time.Ticker
	interval time.Duration
	log      logging.Logger
	stopChan chan struct{}
}

func NewPeriodicExecutor(interval time.Duration, log logging.Logger) *PeriodicExecutor {
	return &PeriodicExecutor{
		ticker:   time.NewTicker(interval),
		interval: interval,
		log:      log,
		stopChan: make(chan struct{}),
	}
}

func (pe *PeriodicExecutor) Start(ctx context.Context, task func(ctx context.Context) error) {
	go func() {
		for {
			select {
			case <-pe.ticker.C:
				pe.log.Tracef("Executing task")
				if err := task(ctx); err != nil {
					pe.log.Errorf("Task execution failed: %v", err)
				}
			case <-pe.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (pe *PeriodicExecutor) Stop() {
	pe.ticker.Stop()
	close(pe.stopChan)
}
