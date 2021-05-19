package swap

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
)

// compile check whether GethClient implements Swap interface
var _ Client = (*GethClient)(nil)

// GethClient manages communication with the Geth node
type GethClient struct {
	httpClient *http.Client // HTTP client must handle authentication implicitly
}

// GethClientOptions holds optional parameters for the GethClient
type GethClientOptions struct {
	HTTPClient *http.Client
}

// NewClient constructs a new Client.
func NewGethClient(baseURL *url.URL, o *GethClientOptions) (c *GethClient) {
	if o == nil {
		o = new(GethClientOptions)
	}

	if o.HTTPClient == nil {
		o.HTTPClient = new(http.Client)
	}

	c = &GethClient{
		httpClient: httpClientWithTransport(baseURL, o.HTTPClient),
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
	From  string
	To    string
	Data  string
	Value string
}

// sendETH makes ETH deposit
func (g *GethClient) SendETH(ctx context.Context, from, to string, ammount int64) (err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, from) {
		return fmt.Errorf("eth account %s not found", from)
	}

	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_sendTransaction",
		Params: []ethRequestParams{{
			From:  from,
			To:    to,
			Value: "0x" + fmt.Sprintf("%x", ammount),
		}},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	err = requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp)

	fmt.Printf("transaction %s from %s to %s ETH %d\n", resp.Result, from, to, ammount)
	return
}

// sendBZZ makes BZZ token deposit
func (g *GethClient) SendBZZ(ctx context.Context, from, to string, ammount int64) (err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, from) {
		return fmt.Errorf("eth account %s not found", from)
	}

	req := ethRequest{
		ID:      "0",
		JsonRPC: "1.0",
		Method:  "eth_sendTransaction",
		Params: []ethRequestParams{{
			From: from,
			To:   to,
			Data: "0x40c10f19" + fmt.Sprintf("%064s", to[2:]) + fmt.Sprintf("%064x", big.NewInt(ammount)),
		}},
	}

	resp := new(struct {
		ID      string `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
	})

	err = requestJSON(ctx, g.httpClient, http.MethodPost, "/", req, &resp)

	fmt.Printf("transaction %s from %s to %s BZZ %d\n", resp.Result, from, to, ammount)
	return
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

func contains(list []string, find string) bool {
	for _, v := range list {
		if v == find {
			return true
		}
	}

	return false
}
