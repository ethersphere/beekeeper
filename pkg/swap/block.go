package swap

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

func (g *GethClient) FetchBlockTime(ctx context.Context) (blockTime int64, err error) {
	latestBlockNumber, err := g.fetchLatestBlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch latest block number: %w", err)
	}

	timestampLatest, err := g.fetchBlockTimestamp(ctx, latestBlockNumber)
	if err != nil {
		return 0, fmt.Errorf("fetch latest block timestamp: %w", err)
	}

	timestampPrevious, err := g.fetchBlockTimestamp(ctx, latestBlockNumber-1)
	if err != nil {
		return 0, fmt.Errorf("fetch previous block timestamp: %w", err)
	}

	return timestampLatest - timestampPrevious, nil
}

type rpcRequest struct {
	ID      string
	JsonRPC string
	Method  string
	Params  []interface{}
}

func (g *GethClient) fetchLatestBlockNumber(ctx context.Context) (int64, error) {
	req := rpcRequest{
		ID:      "1",
		JsonRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	if err := requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return 0, fmt.Errorf("request json: %w", err)
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

	return strconv.ParseInt(resp.Result.Timestamp[2:], 16, 64)
}
