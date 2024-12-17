package stamper

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
)

var _ Client = (*Node)(nil)

type Node struct {
	client *api.Client
	Name   string
}

func NewNodeInfo(client *api.Client, name string) *Node {
	return &Node{
		client: client,
		Name:   name,
	}
}

func (n *Node) Create(ctx context.Context, amount uint64, depth uint8) error {
	_, err := n.client.Postage.CreatePostageBatch(ctx, int64(amount), uint64(depth), "beekeeper")
	if err != nil {
		return fmt.Errorf("node %s: create postage batch: %w", n.Name, err)
	}

	return nil
}

func (n *Node) Dilute(ctx context.Context, threshold float64, depthIncrement uint8) error {
	batches, err := n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get postage batches: %w", n.Name, err)
	}

	for _, batch := range batches {
		if !batch.Usable || batch.ImmutableFlag || batch.Utilization == 0 {
			continue
		}

		usageFactor := batch.Depth - batch.BucketDepth              // depth - bucketDepth
		divisor := float64(int(1) << usageFactor)                   // 2^(depth - bucketDepth)
		stampsUsage := (float64(batch.Utilization) / divisor) * 100 // (utilization / 2^(depth - bucketDepth)) * 100

		if stampsUsage >= threshold {
			newDepth := batch.Depth + depthIncrement
			if err := n.client.Postage.DilutePostageBatch(ctx, batch.BatchID, uint64(newDepth), ""); err != nil {
				return fmt.Errorf("node %s: dilute batch %s: %w", n.Name, batch.BatchID, err)
			}
		}
	}

	return nil
}

func (n *Node) Set(ctx context.Context, ttlThreshold time.Duration, topupDuration time.Duration, threshold float64, depth uint16) error {
	panic("unimplemented")
}

func (n *Node) Topup(ctx context.Context, ttlThreshold time.Duration, topupDuration time.Duration) error {
	panic("unimplemented")
}
