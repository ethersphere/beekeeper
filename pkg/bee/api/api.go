package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper"
)

const (
	apiVersion                  = "v1"
	contentType                 = "application/json, text/plain, */*; charset=utf-8"
	postageStampBatchHeader     = "Swarm-Postage-Batch-Id"
	deferredUploadHeader        = "Swarm-Deferred-Upload"
	swarmAct                    = "Swarm-Act"
	swarmActHistoryAddress      = "Swarm-Act-History-Address"
	swarmActPublisher           = "Swarm-Act-Publisher"
	swarmActTimestamp           = "Swarm-Act-Timestamp"
	swarmPinHeader              = "Swarm-Pin"
	swarmTagHeader              = "Swarm-Tag"
	swarmCacheDownloadHeader    = "Swarm-Cache"
	swarmRedundancyFallbackMode = "Swarm-Redundancy-Fallback-Mode"
)

var userAgent = "beekeeper/" + beekeeper.Version

// Client manages communication with the Bee API.
type Client struct {
	httpClient *http.Client // HTTP client must handle authentication implicitly.
	service    service      // Reuse a single struct instead of allocating one for each service on the heap.

	// Services that API provides.
	Act         *ActService
	Bytes       *BytesService
	Chunks      *ChunksService
	Files       *FilesService
	Dirs        *DirsService
	Pinning     *PinningService
	Tags        *TagsService
	PSS         *PSSService
	SOC         *SOCService
	Stewardship *StewardshipService
	Node        *NodeService
	PingPong    *PingPongService
	Postage     *PostageService
	Stake       *StakingService
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	HTTPClient *http.Client
}

// NewClient constructs a new Client.
func NewClient(baseURL *url.URL, o *ClientOptions) (c *Client) {
	if o == nil {
		o = new(ClientOptions)
	}
	if o.HTTPClient == nil {
		o.HTTPClient = new(http.Client)
	}

	c = newClient(httpClientWithTransport(baseURL, o.HTTPClient))
	return
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all API services.
func newClient(httpClient *http.Client) (c *Client) {
	c = &Client{httpClient: httpClient}
	c.service.client = c
	c.Act = (*ActService)(&c.service)
	c.Bytes = (*BytesService)(&c.service)
	c.Chunks = (*ChunksService)(&c.service)
	c.Files = (*FilesService)(&c.service)
	c.Dirs = (*DirsService)(&c.service)
	c.Pinning = (*PinningService)(&c.service)
	c.Tags = (*TagsService)(&c.service)
	c.PSS = (*PSSService)(&c.service)
	c.SOC = (*SOCService)(&c.service)
	c.Stewardship = (*StewardshipService)(&c.service)
	c.Node = (*NodeService)(&c.service)
	c.PingPong = (*PingPongService)(&c.service)
	c.Postage = (*PostageService)(&c.service)
	c.Stake = (*StakingService)(&c.service)
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

// requestJSON handles the HTTP request response cycle. It JSON encodes the request
// body, creates an HTTP request with provided method on a path with required
// headers and decodes request body if the v argument is not nil and content type is
// application/json.
func (c *Client) requestJSON(ctx context.Context, method, path string, body, v interface{}) (err error) {
	var bodyBuffer io.ReadWriter
	if body != nil {
		bodyBuffer = new(bytes.Buffer)
		if err = encodeJSON(bodyBuffer, body); err != nil {
			return err
		}
	}

	return c.request(ctx, method, path, bodyBuffer, v)
}

// request handles the HTTP request response cycle.
func (c *Client) request(ctx context.Context, method, path string, body io.Reader, v interface{}) (err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)

	r, err := c.httpClient.Do(req)
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

// requestData handles the HTTP request response cycle.
func (c *Client) requestData(ctx context.Context, method, path string, body io.Reader, opts *DownloadOptions) (resp io.ReadCloser, err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)
	// ACT
	if opts != nil {
		if opts.Act != nil {
			req.Header.Set(swarmAct, strconv.FormatBool(*opts.Act))
		}
		if opts.ActHistoryAddress != nil {
			req.Header.Set(swarmActHistoryAddress, (*opts.ActHistoryAddress).String())
		}
		if opts.ActPublicKey != nil {
			req.Header.Set(swarmActPublisher, (*opts.ActPublicKey).String())
		}
		if opts.ActTimestamp != nil {
			req.Header.Set(swarmActTimestamp, strconv.FormatUint(*opts.ActTimestamp, 10))
		}
	}

	if opts != nil && opts.Cache != nil {
		req.Header.Set(swarmCacheDownloadHeader, strconv.FormatBool(*opts.Cache))
	}
	if opts != nil && opts.RedundancyFallbackMode != nil {
		req.Header.Set(swarmRedundancyFallbackMode, strconv.FormatBool(*opts.RedundancyFallbackMode))
	}
	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if err = responseErrorHandler(r); err != nil {
		return nil, err
	}

	return r.Body, nil
}

// requestWithHeader handles the HTTP request response cycle.
func (c *Client) requestWithHeader(ctx context.Context, method, path string, header http.Header, body io.Reader, v interface{}, headerParser ...func(http.Header)) (err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	req.Header = header
	req.Header.Add("Accept", contentType)

	r, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if v != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&v)
		for _, parser := range headerParser {
			parser(r.Header)
		}
		return err
	}

	return err
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

		_, _ = io.Copy(io.Discard, r)
		r.Close()
	}()
}

type messageResponse struct {
	Message string `json:"message"`
}

// responseErrorHandler returns an error based on the HTTP status code or nil if
// the status code is from 200 to 299.
// The error will include the message from standardized JSON-encoded error response
// if it is not the same as the status text.
func responseErrorHandler(r *http.Response) (err error) {
	if r.StatusCode/100 == 2 {
		// no error if response in 2xx range
		return nil
	}

	var e messageResponse
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err = json.NewDecoder(r.Body).Decode(&e); err != nil && err != io.EOF {
			return err
		}
	}

	err = NewHTTPStatusError(r.StatusCode)
	// add message to the error if it is not already the same as the status text
	if e.Message != "" && e.Message != http.StatusText(r.StatusCode) {
		return fmt.Errorf("response message %q: status: %w", e.Message, err)
	}
	return err
}

// service is the base type for all API service providing the Client instance
// for them to use.
type service struct {
	client *Client
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

type UploadOptions struct {
	Act               bool
	Pin               bool
	Tag               uint64
	BatchID           string
	Direct            bool
	ActHistoryAddress swarm.Address
}

type DownloadOptions struct {
	Act                    *bool
	ActHistoryAddress      *swarm.Address
	ActPublicKey           *swarm.Address
	ActTimestamp           *uint64
	Cache                  *bool
	RedundancyFallbackMode *bool
}
