package postage

import (
	"context"
	"fmt"
	mbig "math/big"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents check options
type Options struct {
	GasPrice           string
	PostageAmount      int64
	PostageTopupAmount int64
	PostageDepth       uint64
	PostageNewDepth    uint64
	PostageLabel       string
	NodeCount          int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GasPrice:           "",
		PostageAmount:      1000,
		PostageTopupAmount: 1100,
		PostageDepth:       17,
		PostageNewDepth:    18,
		PostageLabel:       "test-label",
		NodeCount:          1,
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
		return fmt.Errorf("invalid options type")
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.NodeNames()
	node := sortedNodes[0]

	client := clients[node]

	batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel, false)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", node, err)
	}

	batch, err := client.PostageBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed getting postage batch %w", err)
	}

	if batch.Amount.Int64() != o.PostageAmount {
		return fmt.Errorf(
			"create: invalid batch amount, expected %d got %d, batch %s",
			o.PostageAmount,
			batch.Amount.Int64(),
			batchID,
		)
	}
	if batch.Depth != uint8(o.PostageDepth) {
		return fmt.Errorf(
			"create: invalid batch depth, expected %d got %d, batch %s",
			o.PostageDepth,
			batch.Depth,
			batchID,
		)
	}

	c.logger.Infof("node %s: created new batch id %s, amount %d, depth %d", node, batchID, o.PostageAmount, o.PostageDepth)

	c.logger.Infof("node %s: top up with amount %d", node, o.PostageTopupAmount)

	err = client.TopUpPostageBatch(ctx, batchID, o.PostageTopupAmount, o.GasPrice)
	if err != nil {
		return fmt.Errorf("failed topping up batch %s with amount %d gas %s: %w", batchID, o.PostageTopupAmount, o.GasPrice, err)
	}

	batch, err = client.PostageBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed getting postage batch %w", err)
	}

	newAmount := o.PostageAmount + o.PostageTopupAmount

	if batch.Amount.Int64() != newAmount {
		return fmt.Errorf(
			"topup: invalid batch amount, expected %d got %d, batch %s",
			newAmount,
			batch.Amount.Int64(),
			batchID,
		)
	}
	if batch.Depth != uint8(o.PostageDepth) {
		return fmt.Errorf(
			"topup: invalid batch depth, expected %d got %d, batch %s",
			o.PostageDepth,
			batch.Depth,
			batchID,
		)
	}

	c.logger.Infof("node %s: topped up batch id %s", node, batchID)

	depthChange := o.PostageNewDepth - o.PostageDepth

	newValue2 := mbig.NewInt(0).Div(mbig.NewInt(newAmount), mbig.NewInt(int64(1<<depthChange)))

	err = client.DilutePostageBatch(ctx, batchID, o.PostageNewDepth, o.GasPrice)
	if err != nil {
		return fmt.Errorf("failed topping up batch %s with amount %d gas %s: %w", batchID, o.PostageTopupAmount, o.GasPrice, err)
	}

	batch, err = client.PostageBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed getting postage batch %w", err)
	}

	if batch.Amount.Cmp(newValue2) != 0 {
		return fmt.Errorf(
			"dilute: invalid batch amount, expected %d got %d, batch %s",
			newValue2.Int64(),
			batch.Amount.Int64(),
			batchID,
		)
	}
	if batch.Depth != uint8(o.PostageNewDepth) {
		return fmt.Errorf(
			"dilute: invalid batch depth, expected %d got %d, batch %s",
			o.PostageNewDepth,
			batch.Depth,
			batchID,
		)
	}

	c.logger.Infof("node %s: diluted batch id %s", node, batchID)

	return
}
