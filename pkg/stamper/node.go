package stamper

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type node struct {
	client *api.Client
	name   string
	log    logging.Logger
}

func newNodeInfo(client *api.Client, name string, log logging.Logger) *node {
	return &node{
		client: client,
		name:   name,
		log:    log,
	}
}

func (n *node) Create(ctx context.Context, amount uint64, depth uint16) error {
	batchID, err := n.client.Postage.CreatePostageBatch(ctx, int64(amount), uint64(depth), "beekeeper")
	if err != nil {
		return fmt.Errorf("node %s: create postage batch: %w", n.name, err)
	}

	n.log.Infof("node %s: created postage batch %s", n.name, batchID)

	return nil
}

func (n *node) Dilute(ctx context.Context, threshold float64, depthIncrement uint16) error {
	batches, err := n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get postage batches: %w", n.name, err)
	}

	for _, batch := range batches {
		if !batch.Usable || batch.Utilization == 0 {
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

// Set performs both Topup and Dilute operations on postage batches in one function.
// The order of operations is critical because Dilute increases the batch depth,
// which directly affects the calculations for Topup by reducing the effective batch TTL.
// Therefore, Topup is handled first, considering the original depth, followed by Dilute
// which accounts for the new depth and utilization threshold.
func (n *node) Set(ctx context.Context, ttlThreshold time.Duration, topupDuration time.Duration, threshold float64, depth uint16, blockTime int64) error {
	chainState, err := n.client.Postage.GetChainState(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get chain state: %w", n.name, err)
	}

	price := chainState.CurrentPrice.Int64()
	if price <= 0 {
		return fmt.Errorf("node %s: invalid chain price: %d", n.name, price)
	}

	batches, err := n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get postage batches: %w", n.name, err)
	}

	for _, batch := range batches {
		if !batch.Usable || batch.Utilization == 0 {
			continue
		}

		// Topup
		batchTTL := time.Unix(batch.BatchTTL, 0)
		if time.Until(batchTTL) <= ttlThreshold {
			originalDepth := batch.Depth - batch.BucketDepth + uint8(depth)
			adjustedDepth := originalDepth + uint8(depth)
			multiplier := int64(1 << adjustedDepth)

			if blockTime <= 0 {
				blockTime = 1 // avoid division by zero
			}

			remainingTTL := batchTTL.Unix() - time.Now().Unix()
			if remainingTTL < 0 {
				remainingTTL = 0 // handle expired batches
			}

			requiredDuration := int64(topupDuration.Seconds()) - remainingTTL

			if requiredDuration > 0 {
				amount := (requiredDuration / blockTime) * multiplier * price

				if err := n.client.Postage.TopUpPostageBatch(ctx, batch.BatchID, amount, ""); err != nil {
					return fmt.Errorf("node %s: top-up batch %s: %w", n.name, batch.BatchID, err)
				}

				n.log.Infof("node %s: topped up batch %s with amount %d", n.name, batch.BatchID, amount)
			}
		}

		// Dilute logic
		usageFactor := batch.Depth - batch.BucketDepth              // depth - bucketDepth
		divisor := float64(int(1) << usageFactor)                   // 2^(depth - bucketDepth)
		stampsUsage := (float64(batch.Utilization) / divisor) * 100 // (utilization / 2^(depth - bucketDepth)) * 100

		if stampsUsage >= threshold {
			newDepth := uint16(batch.Depth) + depth
			if err := n.client.Postage.DilutePostageBatch(ctx, batch.BatchID, uint64(newDepth), ""); err != nil {
				return fmt.Errorf("node %s: dilute batch %s: %w", n.name, batch.BatchID, err)
			}

			n.log.Infof("node %s: diluted batch %s to depth %d", n.name, batch.BatchID, newDepth)
		}
	}

	return nil
}

func (n *node) Topup(ctx context.Context, ttlThreshold time.Duration, topupDuration time.Duration, blockTime int64) error {
	chainState, err := n.client.Postage.GetChainState(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get chain state: %w", n.name, err)
	}

	price := chainState.CurrentPrice.Int64()
	if price <= 0 {
		return fmt.Errorf("node %s: invalid chain price: %d", n.name, price)
	}

	batches, err := n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return fmt.Errorf("node %s: get postage batches: %w", n.name, err)
	}

	for _, batch := range batches {
		if !batch.Usable || batch.Utilization == 0 {
			continue
		}

		batchTTL := time.Unix(batch.BatchTTL, 0)
		if time.Until(batchTTL) <= ttlThreshold {
			depth := batch.Depth - batch.BucketDepth
			multiplier := int64(1 << depth)

			if blockTime <= 0 {
				blockTime = 1 // avoid division by zero
			}

			secondsToTopup := int64(topupDuration.Seconds())
			timeLeft := batchTTL.Unix() - time.Now().Unix()
			if timeLeft < 0 {
				timeLeft = 0
			}

			requiredDuration := secondsToTopup - timeLeft
			if requiredDuration <= 0 {
				continue
			}

			amount := (requiredDuration / blockTime) * multiplier * price

			if err := n.client.Postage.TopUpPostageBatch(ctx, batch.BatchID, amount, ""); err != nil {
				return fmt.Errorf("node %s: top-up batch %s: %w", n.name, batch.BatchID, err)
			}

			n.log.Infof("node %s: topped up batch %s with amount %d", n.name, batch.BatchID, amount)
		}
	}

	return nil
}
