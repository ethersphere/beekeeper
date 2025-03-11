package smoke

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type BaseCheck struct {
	logger logging.Logger
}

func (b *BaseCheck) RunWithDuration(
	ctx context.Context,
	cluster orchestration.Cluster,
	opts interface{},
	duration time.Duration,
	runFunc func(context.Context, orchestration.Cluster, Options) error,
) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	doneCh := make(chan error, 1)

	go func() {
		doneCh <- runFunc(ctx, cluster, o)
	}()

	select {
	case err := <-doneCh:
		return err
	case <-time.After(duration):
		b.logger.Info("Duration expired, stopping execution")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
