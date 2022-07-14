package swap

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/logging"
)

// compile check whether GethClient implements Swap interface
var _ Client = (*GethClient)(nil)

// GethClient manages communication with the Geth node
type GethClient struct {
	bzzTokenAddress string
	ethAccount      string
	httpClient      *http.Client // HTTP client must handle authentication implicitly
	logger          logging.Logger
}

// GethClientOptions holds optional parameters for the GethClient
type GethClientOptions struct {
	BzzTokenAddress string
	EthAccount      string
	HTTPClient      *http.Client
}

// NewClient constructs a new Client.
func NewGethClient(baseURL *url.URL, o *GethClientOptions, logger logging.Logger) (gc *GethClient) {
	if o == nil {
		o = new(GethClientOptions)
	}

	if o.HTTPClient == nil {
		o.HTTPClient = new(http.Client)
	}

	if len(o.BzzTokenAddress) == 0 {
		o.BzzTokenAddress = BzzTokenAddress
	}

	if len(o.EthAccount) == 0 {
		o.EthAccount = EthAccount
	}

	gc = &GethClient{
		bzzTokenAddress: o.BzzTokenAddress,
		ethAccount:      o.EthAccount,
		httpClient:      httpClientWithTransport(baseURL, o.HTTPClient),
		logger:          logger,
	}

	return
}

// ethRequest represents common eth request
type ethRequest struct {
	ID      string
	JsonRPC string
	Method  string
	Params  []ethRequestParams
}

// ethRequestParams represents common eth request parameters
type ethRequestParams struct {
	From     string
	To       string
	Data     string
	Value    string
	Gas      string
	GasPrice string
}

// SendETH makes ETH deposit
func (g *GethClient) SendETH(ctx context.Context, to string, amount float64) (tx string, err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, g.ethAccount) {
		return "", fmt.Errorf("eth account %s not found", g.ethAccount)
	}

	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_sendTransaction",
		Params: []ethRequestParams{{
			From:     g.ethAccount,
			To:       to,
			Value:    addPrefix("0x", fmt.Sprintf("%x", float64ToBigInt(amount, 1000000000000000000))), // 18 zeroes
			Gas:      addPrefix("0x", fmt.Sprintf("%x", EthGasLimit)),
			GasPrice: addPrefix("0x", fmt.Sprintf("%x", GasPrice)),
		}},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	if err = requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return "", err
	}

	return resp.Result, nil
}

// SendBZZ makes BZZ token deposit
func (g *GethClient) SendBZZ(ctx context.Context, to string, amount float64) (tx string, err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, g.ethAccount) {
		return "", fmt.Errorf("eth account %s not found", g.ethAccount)
	}

	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_sendTransaction",
		Params: []ethRequestParams{{
			From:     g.ethAccount,
			To:       g.bzzTokenAddress,
			Data:     mintBzz + fmt.Sprintf("%064s", strings.TrimPrefix(to, "0x")) + fmt.Sprintf("%064x", float64ToBigInt(amount, 10000000000000000)), // 16 zeroes
			Gas:      addPrefix("0x", fmt.Sprintf("%x", BzzGasLimit)),
			GasPrice: addPrefix("0x", fmt.Sprintf("%x", GasPrice)),
		}},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	if err = requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return "", err
	}

	return resp.Result, nil
}

// SendGBZZ makes gBZZ token deposit
func (g *GethClient) SendGBZZ(ctx context.Context, to string, amount float64) (tx string, err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, g.ethAccount) {
		return "", fmt.Errorf("eth account %s not found", g.ethAccount)
	}

	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_sendTransaction",
		Params: []ethRequestParams{{
			From:     g.ethAccount,
			To:       g.bzzTokenAddress,
			Data:     transferBzz + fmt.Sprintf("%064s", strings.TrimPrefix(to, "0x")) + fmt.Sprintf("%064x", float64ToBigInt(amount, 10000000000000000)), // 16 zeroes
			Gas:      addPrefix("0x", fmt.Sprintf("%x", BzzGasLimit)),
			GasPrice: addPrefix("0x", fmt.Sprintf("%x", GasPrice)),
		}},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	if err = requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return "", err
	}

	return resp.Result, nil
}

func (g *GethClient) AttestOverlayEthAddress(ctx context.Context, ethAddr string) (tx string, err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return "", fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, g.ethAccount) {
		return "", fmt.Errorf("eth account %s not found", g.ethAccount)
	}

	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_sendTransaction",
		Params: []ethRequestParams{{
			From:     g.ethAccount,
			To:       g.ethAccount,
			Data:     fmt.Sprintf("0x%064s", ethAddr),
			Gas:      addPrefix("0x", fmt.Sprintf("%x", BzzGasLimit)),
			GasPrice: addPrefix("0x", fmt.Sprintf("%x", GasPrice)),
		}},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	if err = requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp); err != nil {
		return "", err
	}

	return resp.Result, nil
}

// ethAccounts returns list of accounts
func (g *GethClient) ethAccounts(ctx context.Context) (a []string, err error) {
	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_accounts",
		Params:  []ethRequestParams{},
	}

	resp := new(struct {
		ID      string   `json:"id"`
		JsonRPC string   `json:"jsonrpc"`
		Result  []string `json:"result"`
	})

	if err := requestJSON(ctx, g.httpClient, http.MethodGet, "/", req, &resp); err != nil {
		return nil, err
	}

	return resp.Result, nil
}

// contains checks if list contains string
func contains(list []string, find string) bool {
	for _, v := range list {
		if v == find {
			return true
		}
	}

	return false
}

// addPrefix adds prefix to string if it doesn't exist
func addPrefix(prefix, to string) string {
	if !strings.HasPrefix(to, prefix) {
		return prefix + to
	}
	return to
}

// float64ToBigInt converts float64 to big.Int
func float64ToBigInt(f float64, coin int64) *big.Int {
	bigFloat := new(big.Float)
	bigFloat.SetFloat64(f)

	bigCoin := new(big.Float)
	bigCoin.SetInt64(coin)

	bigFloat.Mul(bigFloat, bigCoin)

	result := new(big.Int)
	bigFloat.Int(result) // store converted number in result

	return result
}
