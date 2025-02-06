package swap

import (
	"context"
	"errors"
	"fmt"
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
	offset int64
}

func WithOffset(offset int64) Option {
	return func(o *options) {
		if offset > 0 {
			o.offset = offset
		} else {
			o.offset = 1
		}
	}
}

func (g *GethClient) FetchBlockTime(ctx context.Context, opts ...Option) (int64, error) {
	o := processOptions(opts...)

	retryOffset := o.offset

	var timestampLatest, timestampPrevious int64
	var err error

	latestBlockNumber, err := g.fetchLatestBlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch latest block number: %w", err)
	}

	timestampLatest, err = g.fetchBlockTimestamp(ctx, latestBlockNumber)
	if err != nil {
		return 0, fmt.Errorf("fetch latest block timestamp: %w", err)
	}

	for retryOffset >= 1 {
		// enusure that blockNumber is between 1 and latestBlockNumber
		if retryOffset > latestBlockNumber {
			retryOffset = latestBlockNumber - 1
		}
		blockNumber := latestBlockNumber - retryOffset

		timestampPrevious, err = g.fetchBlockTimestamp(ctx, blockNumber)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrEmptyTimestamp) || retryOffset == 1 {
			return 0, fmt.Errorf("fetch previous block timestamp (block %d): %w", blockNumber, err)
		}

		// if not the first iteration, reduce the offset by half and retry
		retryOffset /= 2
		g.logger.Warningf("%v at at block %d, offset reduced to %d and retrying...", err, blockNumber, retryOffset)
	}

	blockTime := (timestampLatest - timestampPrevious) / retryOffset

	g.logger.Tracef("block time: %d seconds", blockTime)

	return blockTime, nil
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

	if err := requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
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

	if err := requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
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
		offset: 1,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
