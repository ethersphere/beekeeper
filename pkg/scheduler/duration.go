package scheduler

import (
	"context"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
)

// DurationExecutor executes a task and then stops gracefully after a specified duration.
type DurationExecutor struct {
	duration time.Duration
	log      logging.Logger
}

// NewDurationExecutor creates a new DurationExecutor with the given duration and logger.
func NewDurationExecutor(duration time.Duration, log logging.Logger) *DurationExecutor {
	return &DurationExecutor{
		duration: duration,
		log:      log,
	}
}

// Run executes the given task and waits for the specified duration before stopping.
func (de *DurationExecutor) Run(ctx context.Context, task func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	doneCh := make(chan error, 1)
	defer close(doneCh)

	go func() {
		doneCh <- task(ctx)
	}()

	select {
	case err := <-doneCh:
		return err
	case <-time.After(de.duration):
		de.log.Infof("Duration of %s expired, stopping executor", de.duration)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
