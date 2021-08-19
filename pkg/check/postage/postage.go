package postage

import (
	"context"
	"fmt"
	mbig "math/big"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
)

// Options represents check options
type Options struct {
	GasPrice           string
	PostageAmount      int64
	PostageTopupAmount int64
	PostageDepth       uint64
	PostageNewDepth    uint64
	PostageLabel       string
	PostageWait        time.Duration
	NodeCount          int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GasPrice:           "",
		PostageAmount:      1000,
		PostageTopupAmount: 100,
		PostageDepth:       17,
		PostageNewDepth:    18,
		PostageLabel:       "test-label",
		PostageWait:        5 * time.Second,
		NodeCount:          1,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
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
	time.Sleep(o.PostageWait)

	batches, err := client.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed getting postage batches %w", err)
	}

	found := false
	for _, v := range batches {
		if v.BatchID == batchID {
			found = true
			if v.Amount.Int64() != o.PostageAmount {
				return fmt.Errorf(
					"invalid batch amount expected %d got %d, batch %s",
					o.PostageAmount,
					v.Amount.Int64(),
					batchID,
				)
			}
			if v.Depth != uint8(o.PostageDepth) {
				return fmt.Errorf(
					"invalid batch amount expected %d got %d, batch %s",
					o.PostageDepth,
					v.Depth,
					batchID,
				)
			}
		}
	}
	if !found {
		return fmt.Errorf("cannot find batch %s", batchID)
	}

	fmt.Printf("node %s: created new batch id %s\n", node, batchID)

	err = client.TopUpPostageBatch(ctx, batchID, o.PostageTopupAmount, o.GasPrice)
	if err != nil {
		return fmt.Errorf("failed topping up batch %s with amount %d gas %s: %w", batchID, o.PostageTopupAmount, o.GasPrice, err)
	}
	time.Sleep(o.PostageWait)

	batches, err = client.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed getting postage batches %w", err)
	}

	newAmount := o.PostageAmount + o.PostageTopupAmount

	found = false
	for _, v := range batches {
		if v.BatchID == batchID {
			found = true
			if v.Amount.Int64() != newAmount {
				return fmt.Errorf(
					"invalid batch amount expected %d got %d, batch %s",
					newAmount,
					v.Amount.Int64(),
					batchID,
				)
			}
			if v.Depth != uint8(o.PostageDepth) {
				return fmt.Errorf(
					"invalid batch amount expected %d got %d, batch %s",
					o.PostageDepth,
					v.Depth,
					batchID,
				)
			}
		}
	}
	if !found {
		return fmt.Errorf("cannot find batch %s", batchID)
	}

	fmt.Printf("node %s: topped up batch id %s\n", node, batchID)

	depthChange := o.PostageNewDepth - o.PostageDepth

	newValue2 := mbig.NewInt(0).Div(mbig.NewInt(newAmount), mbig.NewInt(int64(1<<depthChange)))

	err = client.DilutePostageBatch(ctx, batchID, o.PostageNewDepth, o.GasPrice)
	if err != nil {
		return fmt.Errorf("failed topping up batch %s with amount %d gas %s: %w", batchID, o.PostageTopupAmount, o.GasPrice, err)
	}
	time.Sleep(o.PostageWait)

	batches, err = client.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed getting postage batches %w", err)
	}

	found = false
	for _, v := range batches {
		if v.BatchID == batchID {
			found = true
			if v.Amount.Cmp(newValue2) != 0 {
				return fmt.Errorf(
					"invalid batch amount expected %d got %d, batch %s",
					newValue2.Int64(),
					v.Amount.Int64(),
					batchID,
				)
			}
			if v.Depth != uint8(o.PostageNewDepth) {
				return fmt.Errorf(
					"invalid batch amount expected %d got %d, batch %s",
					o.PostageNewDepth,
					v.Depth,
					batchID,
				)
			}
		}
	}
	if !found {
		return fmt.Errorf("cannot find batch %s", batchID)
	}

	fmt.Printf("node %s: diluted batch id %s\n", node, batchID)

	return
}
