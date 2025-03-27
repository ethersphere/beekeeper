package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
)

// DurationExecutor executes a task and then stops immediately after the specified duration.
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
	if task == nil {
		return errors.New("task cannot be nil")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	doneCh := make(chan error, 1)

	go func() {
		select {
		case <-ctx.Done():
		case doneCh <- task(ctx):
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(de.duration):
		de.log.Infof("Duration of %s expired, stopping executor", de.duration)
		return nil
	case err := <-doneCh:
		return err
	}
}
