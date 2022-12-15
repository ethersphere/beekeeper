package stake

import (
	"context"
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
		return err
	}
	withdrawStake, err := client.WithdrawStake(ctx)
	if err != nil {
		return err
	}

	if withdrawStake.Cmp(o.Amount) == 0 {
		return fmt.Errorf("expected withdraw stake to be %v, got %v", o.Amount, withdrawStake)
	}

	_, err = client.DepositStake(ctx, o.InsufficientAmount)

	if !debugapi.IsHTTPStatusErrorCode(err, 400) {
		return fmt.Errorf("expected code %v, got %v", 400, err)
	}

	return nil
}
