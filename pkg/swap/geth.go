package swap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethersphere/beekeeper"
)

const contentType = "application/json; charset=utf-8"

var userAgent = "beekeeper/" + beekeeper.Version

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

	return newGethClient(httpClientWithTransport(baseURL, o.HTTPClient))
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all API services.
func newGethClient(httpClient *http.Client) (c *GethClient) {
	c = &GethClient{httpClient: httpClient}
	return c
}

func httpClientWithTransport(baseURL *url.URL, c *http.Client) *http.Client {
	if c == nil {
		c = new(http.Client)
	}

	transport := c.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}

	c.Transport = roundTripperFunc(func(r *http.Request) (resp *http.Response, err error) {
		r.Header.Set("User-Agent", userAgent)
		u, err := baseURL.Parse(r.URL.String())
		if err != nil {
			return nil, err
		}
		r.URL = u
		return transport.RoundTrip(r)
	})
	return c
}

func (g *GethClient) Fund(ctx context.Context, address string, ethAccount, bzzTokenAddress string, ethDeposit, bzzDeposit int64) (err error) {
	ethAccounts, err := g.ethAccounts(ctx)
	if err != nil {
		return fmt.Errorf("get accounts: %w", err)
	}

	if !contains(ethAccounts, ethAccount) {
		return fmt.Errorf("eth account %s not found", ethAccount)
	}

	if ethDeposit > 0 {
		if err := g.sendETH(ctx, ethAccount, address, ethDeposit); err != nil {
			return fmt.Errorf("send eth: %w", err)
		}
	}

	if bzzDeposit > 0 {
		if err := g.sendBZZ(ctx, ethAccount, bzzTokenAddress, bzzDeposit); err != nil {
			return fmt.Errorf("deposit bzz: %w", err)
		}
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

	if err := g.requestJSON(ctx, http.MethodGet, "/", req, &resp); err != nil {
		return nil, err
	}

	return resp.Result, nil
}

// sendETH makes ETH deposit
func (g *GethClient) sendETH(ctx context.Context, from, to string, ammount int64) (err error) {
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

	err = g.requestJSON(ctx, http.MethodPost, "/", req, &resp)

	fmt.Printf("transaction %s from %s to %s ETH %d\n", resp.Result, from, to, ammount)
	return
}

// sendBZZ makes BZZ token deposit
func (g *GethClient) sendBZZ(ctx context.Context, from, to string, ammount int64) (err error) {
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

	err = g.requestJSON(ctx, http.MethodPost, "/", req, &resp)

	fmt.Printf("transaction %s from %s to %s BZZ %d\n", resp.Result, from, to, ammount)
	return
}

// requestJSON handles the HTTP request response cycle. It JSON encodes the request
// body, creates an HTTP request with provided method on a path with required
// headers and decodes request body if the v argument is not nil and content type is
// application/json.
func (g *GethClient) requestJSON(ctx context.Context, method, path string, body, v interface{}) (err error) {
	var bodyBuffer io.ReadWriter
	if body != nil {
		bodyBuffer = new(bytes.Buffer)
		if err = encodeJSON(bodyBuffer, body); err != nil {
			return err
		}
	}

	return g.request(ctx, method, path, bodyBuffer, v)
}

// request handles the HTTP request response cycle.
func (g *GethClient) request(ctx context.Context, method, path string, body io.Reader, v interface{}) (err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)

	r, err := g.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer drain(r.Body)

	if err = responseErrorHandler(r); err != nil {
		return err
	}

	if v != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return json.NewDecoder(r.Body).Decode(&v)
	}
	return nil
}

// encodeJSON writes a JSON-encoded v object to the provided writer with
// SetEscapeHTML set to false.
func encodeJSON(w io.Writer, v interface{}) (err error) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// drain discards all of the remaining data from the reader and closes it,
// asynchronously.
func drain(r io.ReadCloser) {
	go func() {
		// Panicking here does not put data in
		// an inconsistent state.
		defer func() {
			_ = recover()
		}()

		_, _ = io.Copy(ioutil.Discard, r)
		r.Close()
	}()
}

// responseErrorHandler returns an error based on the HTTP status code or nil if
// the status code is from 200 to 299.
func responseErrorHandler(r *http.Response) (err error) {
	if r.StatusCode/100 == 2 {
		return nil
	}
	switch r.StatusCode {
	case http.StatusBadRequest:
		return decodeBadRequest(r)
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrTooManyRequests
	case http.StatusInternalServerError:
		return ErrInternalServerError
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return errors.New(strings.ToLower(r.Status))
	}
}

// decodeBadRequest parses the body of HTTP response that contains a list of
// errors as the result of bad request data.
func decodeBadRequest(r *http.Response) (err error) {

	type badRequestResponse struct {
		Errors []string `json:"errors"`
	}

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return NewBadRequestError("bad request")
	}
	var e badRequestResponse
	if err = json.NewDecoder(r.Body).Decode(&e); err != nil {
		if err == io.EOF {
			return NewBadRequestError("bad request")
		}
		return err
	}
	return NewBadRequestError(e.Errors...)
}

// Bool is a helper routine that allocates a new bool value to store v and
// returns a pointer to it.
func Bool(v bool) (p *bool) { return &v }

// roundTripperFunc type is an adapter to allow the use of ordinary functions as
// http.RoundTripper interfaces. If f is a function with the appropriate
// signature, roundTripperFunc(f) is a http.RoundTripper that calls f.
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f(r).
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func contains(list []string, find string) bool {
	for _, v := range list {
		if v == find {
			return true
		}
	}

	return false
}
