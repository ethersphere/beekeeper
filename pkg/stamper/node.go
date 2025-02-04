package stamper

import (
	"context"
	"fmt"
	"slices"
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

func (n *node) Create(ctx context.Context, duration time.Duration, depth uint16, postageLabel string, secondsPerBlock int64) error {
	price, err := n.getPrice(ctx)
	if err != nil {
		return err
	}

	amount := (int64(duration.Seconds()) / secondsPerBlock) * price

	batchID, err := n.client.Postage.CreatePostageBatch(ctx, amount, uint64(depth), postageLabel)
	if err != nil {
		return fmt.Errorf("node %s: create postage batch: %w", n.name, err)
	}

	totalPrice := amount * int64(1<<depth) / (10 ^ 16)

	n.log.WithField("totalPriceBZZ", totalPrice).Infof("node %s: created postage batch %s", n.name, batchID)

	return nil
}

func (n *node) Dilute(ctx context.Context, threshold float64, depthIncrement uint16, opts *options) error {
	batches, _, err := n.getPostageBatches(ctx, false)
	if err != nil {
		return err
	}

	for _, batch := range batches {
		if !isValidBatch(&batch, opts) {
			continue
		}

		if batch.BatchUsage() >= threshold {
			return n.handleDilution(ctx, batch, depthIncrement)
		}
	}

	return nil
}

// Set performs Topup and Dilute operations on postage batches.
// Topup is handled first based on the original depth, followed by Dilute
// which considers the new depth and utilization threshold.
func (n *node) Set(
	ctx context.Context,
	ttlThreshold time.Duration,
	topUpFinalTTL time.Duration,
	utilizationThreshold float64,
	extraDepth uint16,
	secondsPerBlock int64,
	opts *options,
) error {
	batches, price, err := n.getPostageBatches(ctx, true)
	if err != nil {
		return err
	}

	for _, batch := range batches {
		if !isValidBatch(&batch, opts) {
			continue
		}

		batchTTL := time.Duration(batch.BatchTTL) * time.Second

		needsDilution := batch.BatchUsage() >= utilizationThreshold

		if needsDilution {
			batchTTL = batchTTL / (1 << extraDepth) // reduce batch TTL by 2^extraDepth
		}

		if batchTTL > ttlThreshold && !needsDilution {
			continue
		}

		if err := n.handleTopup(ctx, batch, ttlThreshold, topUpFinalTTL, batchTTL, secondsPerBlock, price); err != nil {
			return err
		}

		if needsDilution {
			return n.handleDilution(ctx, batch, extraDepth)
		}
	}

	return nil
}

func (n *node) Topup(ctx context.Context, ttlThreshold time.Duration, topUpFinalTTL time.Duration, secondsPerBlock int64, opts *options) error {
	batches, price, err := n.getPostageBatches(ctx, true)
	if err != nil {
		return err
	}

	for _, batch := range batches {
		if !isValidBatch(&batch, opts) {
			continue
		}

		batchTTL := time.Duration(batch.BatchTTL) * time.Second

		return n.handleTopup(ctx, batch, ttlThreshold, topUpFinalTTL, batchTTL, secondsPerBlock, price)
	}

	return nil
}

func (n *node) handleDilution(ctx context.Context, batch api.PostageStampResponse, extraDepth uint16) error {
	newDepth := uint16(batch.Depth) + extraDepth

	n.log.Tracef("node %s: batch %s: usage %.2f%%, diluting to depth %d", n.name, batch.BatchID, batch.BatchUsage(), newDepth)

	if err := n.client.Postage.DilutePostageBatch(ctx, batch.BatchID, uint64(newDepth), ""); err != nil {
		return fmt.Errorf("node %s: dilute batch %s: %w", n.name, batch.BatchID, err)
	}

	n.log.Infof("node %s: diluted batch %s to depth %d", n.name, batch.BatchID, newDepth)

	return nil
}

func (n *node) handleTopup(ctx context.Context, batch api.PostageStampResponse, ttlThreshold, topUpFinalTTL, batchTTL time.Duration, secondsPerBlock, price int64) error {
	if batchTTL <= ttlThreshold {
		topUpTTL := topUpFinalTTL - batchTTL
		if topUpTTL > 0 {
			amount := (int64(topUpTTL.Seconds()) / secondsPerBlock) * price

			n.log.Tracef("node %s: batch %s: required duration %d, amount %d", n.name, batch.BatchID, topUpTTL, amount)

			if err := n.client.Postage.TopUpPostageBatch(ctx, batch.BatchID, amount, ""); err != nil {
				return fmt.Errorf("node %s: top-up batch %s: %w", n.name, batch.BatchID, err)
			}

			n.log.Infof("node %s: topped up batch %s with amount %d", n.name, batch.BatchID, amount)
		}
	}

	return nil
}

func (n *node) getPrice(ctx context.Context) (int64, error) {
	chainState, err := n.client.Postage.GetChainState(ctx)
	if err != nil {
		return 0, fmt.Errorf("node %s: get chain state: %w", n.name, err)
	}

	price := chainState.CurrentPrice.Int64()
	if price <= 0 {
		return 0, fmt.Errorf("node %s: invalid chain price: %d", n.name, price)
	}

	return price, nil
}

func (n *node) getPostageBatches(ctx context.Context, needPrice bool) (batches []api.PostageStampResponse, price int64, err error) {
	if needPrice {
		price, err = n.getPrice(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("node %s: get price: %w", n.name, err)
		}
	}

	batches, err = n.client.Postage.PostageBatches(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("node %s: get postage batches: %w", n.name, err)
	}

	return batches, price, nil
}

// isValidBatch checks if a batch should be processed
func isValidBatch(batch *api.PostageStampResponse, opts *options) bool {
	if !batch.Usable || batch.Utilization == 0 || batch.BatchTTL <= 0 {
		return false
	}

	if len(opts.batchIDs) > 0 && !slices.Contains(opts.batchIDs, batch.BatchID) {
		return false
	}

	if len(opts.postageLabels) > 0 && !slices.Contains(opts.postageLabels, batch.Label) {
		return false
	}

	return true
}
