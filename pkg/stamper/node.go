package stamper

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

var _ Client = (*Node)(nil)

type Node struct {
	client *api.Client
	name   string
	log    logging.Logger
}

func NewNodeInfo(client *api.Client, name string, log logging.Logger) *Node {
	return &Node{
		client: client,
		name:   name,
		log:    log,
	}
}

func (n *Node) Create(ctx context.Context, amount uint64, depth uint16) error {
	batchID, err := n.client.Postage.CreatePostageBatch(ctx, int64(amount), uint64(depth), "beekeeper")
	if err != nil {
		return fmt.Errorf("node %s: create postage batch: %w", n.name, err)
	}

	n.log.Infof("node %s: created postage batch %s", n.name, batchID)

	return nil
}

func (n *Node) Dilute(ctx context.Context, threshold float64, depthIncrement uint16) error {
	batches, err := n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get postage batches: %w", n.name, err)
	}

	for _, batch := range batches {
		if !batch.Usable || batch.ImmutableFlag || batch.Utilization == 0 {
			continue
		}

		usageFactor := batch.Depth - batch.BucketDepth              // depth - bucketDepth
		divisor := float64(int(1) << usageFactor)                   // 2^(depth - bucketDepth)
		stampsUsage := (float64(batch.Utilization) / divisor) * 100 // (utilization / 2^(depth - bucketDepth)) * 100

		if stampsUsage >= threshold {
			newDepth := uint16(batch.Depth) + depthIncrement
			if err := n.client.Postage.DilutePostageBatch(ctx, batch.BatchID, uint64(newDepth), ""); err != nil {
				return fmt.Errorf("node %s: dilute batch %s: %w", n.name, batch.BatchID, err)
			}

			n.log.Infof("node %s: diluted batch %s to depth %d", n.name, batch.BatchID, newDepth)
		}
	}

	return nil
}

func (n *Node) Set(ctx context.Context, ttlThreshold time.Duration, topupDuration time.Duration, threshold float64, depth uint16) error {
	if err := n.Dilute(ctx, threshold, depth); err != nil {
		return fmt.Errorf("node %s: dilute: %w", n.name, err)
	}

	if err := n.Topup(ctx, ttlThreshold, topupDuration); err != nil {
		return fmt.Errorf("node %s: topup: %w", n.name, err)
	}

	return nil
}

func (n *Node) Topup(ctx context.Context, ttlThreshold time.Duration, topupDuration time.Duration) error {
	batches, err := n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get postage batches: %w", n.name, err)
	}

	for _, batch := range batches {
		if !batch.Usable || batch.ImmutableFlag || batch.Utilization == 0 {
			continue
		}

		batchTTL := time.Unix(batch.BatchTTL, 0)
		if time.Until(batchTTL) <= ttlThreshold {
			// TODO: calculate amount to topup based on topupDuration
			if err := n.client.Postage.TopUpPostageBatch(ctx, batch.BatchID, 10000, ""); err != nil {
				return fmt.Errorf("node %s: topup batch %s: %w", n.name, batch.BatchID, err)
			}

			n.log.Infof("node %s: topped up batch %s", n.name, batch.BatchID)
		}
	}

	return nil
}
