package stake

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents stake options
type Options struct {
	Amount             *big.Int
	InsufficientAmount *big.Int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Amount:             big.NewInt(100000000000000000),
		InsufficientAmount: big.NewInt(102400),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return err
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.FullNodeNames()
	node := sortedNodes[1]
	c.logger.Infof("checking stake for node %s", node)
	client := clients[node]

	_, err = client.DepositStake(ctx, o.Amount)
	if err != nil {
		return fmt.Errorf("initial stake deposit: %w", err)
	}

	currentStake, err := client.GetStake(ctx)
	if err != nil {
		return fmt.Errorf("get initial stake amount: %w", err)
	}

	if currentStake.Cmp(o.Amount) == 0 {
		return fmt.Errorf("expected stake amount to be %v, got %v", o.Amount, currentStake)
	}

	// can not stake less than previously staked
	_, err = client.DepositStake(ctx, new(big.Int).Sub(o.Amount, big.NewInt(1)))
	if err == nil {
		return errors.New("expected deposit stake to fail")
	}

	// should allow depositing more
	_, err = client.DepositStake(ctx, new(big.Int).Add(o.Amount, big.NewInt(1)))
	if err != nil {
		return fmt.Errorf("increase stake amount: %w", err)
	}

	_, err = client.WithdrawStake(ctx)
	if err == nil {
		return errors.New("expected error on withdraw from running contract")
	}

	_, err = client.DepositStake(ctx, o.InsufficientAmount)

	if !debugapi.IsHTTPStatusErrorCode(err, 400) {
		return fmt.Errorf("deposit insufficient stake amount: expected code %v, got %v", 400, err)
	}

	return nil
}
