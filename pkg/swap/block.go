package swap

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
)

var (
	ErrEmptyTimestamp   = errors.New("empty timestamp, offset too large")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrEmptyResult      = errors.New("empty result")
	ErrInvalidResult    = errors.New("invalid result")
)

type Option func(*options)

type options struct {
	offset  int64
	refresh bool
}

// WithOffset sets the number of blocks to use for block time estimation.
func WithOffset(offset int64) Option {
	return func(o *options) {
		if offset > 0 {
			o.offset = offset
		} else {
			o.offset = 1
		}
	}
}

// WithRefresh forces the block time to be recalculated.
func WithRefresh() Option {
	return func(o *options) {
		o.refresh = true
	}
}

// FetchBlockTime estimates the average block time by comparing timestamps
// of the latest block and an earlier block, adjusting the offset if needed.
// The block time is cached and reused until forced to refresh.
func (g *GethClient) FetchBlockTime(ctx context.Context, opts ...Option) (int64, error) {
	o := processOptions(opts...)

	retryOffset := o.offset

	var err error

	// return cached block time if available and not forced to refresh
	if cachedBlockTime := g.cache.BlockTime(); cachedBlockTime > 0 && !o.refresh {
		return cachedBlockTime, nil
	}

	latestBlockNumber, err := g.fetchLatestBlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch latest block number: %w", err)
	}

	timestampLatest, err := g.fetchBlockTimestamp(ctx, latestBlockNumber)
	if err != nil {
		return 0, fmt.Errorf("fetch latest block timestamp: %w", err)
	}

	// limit retryOffset to at most half of the latest block number
	if retryOffset > latestBlockNumber/2 {
		retryOffset = latestBlockNumber / 2
		g.logger.Warningf("offset too large, reduced to %d", retryOffset)
	}

	var timestampPrevious int64

	for retryOffset >= 1 {
		blockNumber := latestBlockNumber - retryOffset
		timestampPrevious, err = g.fetchBlockTimestamp(ctx, blockNumber)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrEmptyTimestamp) || retryOffset == 1 {
			return 0, fmt.Errorf("fetch previous block timestamp (block %d): %w", blockNumber, err)
		}

		// reduce offset for next attempt, ensuring it remains >= 1
		retryOffset = int64(math.Max(1, float64(retryOffset)/2))
		g.logger.Warningf("%v at block %d, offset reduced to %d and retrying...", err, blockNumber, retryOffset)
	}

	blockTime := float64(timestampLatest-timestampPrevious) / float64(retryOffset)
	roundedBlockTime := int64(math.Round(blockTime))

	g.logger.Tracef("avg block time for last %d blocks: %f, using rounded seconds: %d", retryOffset, blockTime, roundedBlockTime)

	g.cache.SetBlockTime(roundedBlockTime)

	return roundedBlockTime, nil
}

type rpcRequest struct {
	ID      string        `json:"id"`
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

func (g *GethClient) fetchLatestBlockNumber(ctx context.Context) (int64, error) {
	req := rpcRequest{
		JsonRPC: "2.0",
		Method:  "eth_blockNumber",
		ID:      "1",
	}

	resp := new(struct {
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
		ID      string `json:"id"`
	})

	if err := g.requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return 0, fmt.Errorf("request json: %w", err)
	}

	if len(resp.Result) == 0 {
		return 0, ErrEmptyResult
	}

	if resp.Result[:2] != "0x" {
		return 0, ErrInvalidResult
	}

	blockNumber, err := strconv.ParseInt(resp.Result[2:], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parse int: %w", err)
	}

	return blockNumber, nil
}

func (g *GethClient) fetchBlockTimestamp(ctx context.Context, blockNumber int64) (int64, error) {
	req := rpcRequest{
		ID:      "1",
		JsonRPC: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{fmt.Sprintf("0x%x", blockNumber), false},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  struct {
			Timestamp string `json:"timestamp"`
		} `json:"result"`
	})

	if err := g.requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return 0, fmt.Errorf("request json: %w", err)
	}

	if len(resp.Result.Timestamp) == 0 {
		return 0, ErrEmptyTimestamp
	}

	if resp.Result.Timestamp[:2] != "0x" {
		return 0, ErrInvalidTimestamp
	}

	return strconv.ParseInt(resp.Result.Timestamp[2:], 16, 64)
}

func processOptions(opts ...Option) *options {
	o := &options{
		offset:  1,
		refresh: false,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
