package withdraw

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	test "github.com/ethersphere/beekeeper/pkg/test"
)

// Options represents check options
type Options struct {
	TargetAddr string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	logger logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	var checkCase *test.CheckCase

	if checkCase, err = test.NewCheckCase(ctx, cluster, test.CaseOptions{}, c.logger); err != nil {
		return err
	}

	target := checkCase.Bee(1)

	c.logger.Infof("target is %s", target.Name())
	c.logger.Info("withdrawing bzz...")

	if err := target.Withdraw(ctx, "BZZ", o.TargetAddr); err != nil {
		return fmt.Errorf("withdraw bzz: %w", err)
	}

	c.logger.Info("success")
	c.logger.Info("withdrawing native...")

	if err := target.Withdraw(ctx, "xDAI", o.TargetAddr); err != nil {
		return fmt.Errorf("withdraw native: %w", err)
	}

	c.logger.Info("success")
	c.logger.Info("withdrawing to a non whitelisted address")

	var zeroAddr common.Address

	if err := target.Withdraw(ctx, "sETH", zeroAddr.String()); err == nil {
		return errors.New("withdraw to non-whitelisted address expected to fail")
	}

	c.logger.Info("success")

	return nil
}
