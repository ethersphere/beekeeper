package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethersphere/beekeeper"
)

const (
	apiVersion              = "v1"
	contentType             = "application/json; charset=utf-8"
	postageStampBatchHeader = "Swarm-Postage-Batch-Id"
)

var userAgent = "beekeeper/" + beekeeper.Version

// Client manages communication with the Bee API.
type Client struct {
	httpClient *http.Client // HTTP client must handle authentication implicitly.
	service    service      // Reuse a single struct instead of allocating one for each service on the heap.

	// Services that API provides.
	Bytes       *BytesService
	Chunks      *ChunksService
	Files       *FilesService
	Dirs        *DirsService
	Pinning     *PinningService
	Tags        *TagsService
	PSS         *PSSService
	SOC         *SOCService
	Stewardship *StewardshipService
	Auth        *AuthService
}

// Authenticator retrieves the security token
type Authenticator interface {
	Authenticate(context.Context, string) (string, error)
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	HTTPClient *http.Client
	Restricted bool
	Username   string
	Password   string
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

	c.service.username = o.Username
	c.service.password = o.Password
	c.service.restricted = o.Restricted

	return
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all API services.
func newClient(httpClient *http.Client) (c *Client) {
	c = &Client{httpClient: httpClient}
	c.service.client = c
	c.Bytes = (*BytesService)(&c.service)
	c.Chunks = (*ChunksService)(&c.service)
	c.Files = (*FilesService)(&c.service)
	c.Dirs = (*DirsService)(&c.service)
	c.Pinning = (*PinningService)(&c.service)
	c.Tags = (*TagsService)(&c.service)
	c.PSS = (*PSSService)(&c.service)
	c.SOC = (*SOCService)(&c.service)
	c.Stewardship = (*StewardshipService)(&c.service)
	c.Auth = (*AuthService)(&c.service)
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

	r, err := c.requestData(ctx, method, path, nil, bodyBuffer, v)
	if err != nil {
		return err
	}

	defer drain(r.Body)

	if v != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return json.NewDecoder(r.Body).Decode(&v)
	}

	return nil
}

// requestWithHeader handles the HTTP request response cycle.
func (c *Client) requestWithHeader(ctx context.Context, method, path string, header http.Header, body io.Reader, v interface{}) (err error) {
	r, err := c.requestData(ctx, method, path, header, body, v)
	if err != nil {
		return err
	}

	if v != nil && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&v)
		return err
	}

	return err
}

// encodeJSON writes a JSON-encoded v object to the provided writer with
// SetEscapeHTML set to false.
func encodeJSON(w io.Writer, v interface{}) (err error) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// requestData handles the HTTP request response cycle.
func (c *Client) requestData(ctx context.Context, method, path string, header http.Header, body io.Reader, v interface{}) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	if header != nil {
		req.Header = header
	}

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)

	if c.service.restricted {
		role := GetRole(method, path)
		if role == "" {
			return nil, fmt.Errorf("role not found for %s %s", method, path)
		}
		key, err := c.Auth.Authenticate(ctx, role, c.service.username, c.service.password)
		if err != nil {
			return nil, fmt.Errorf("authenticate: %w", err)
		} else {
			bearer := fmt.Sprintf("Bearer %s", key)
			req.Header.Set("Authorization", bearer)
		}
	}

	r, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if err = responseErrorHandler(r); err != nil {
		return nil, err
	}

	return r, nil
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

	restricted bool
	username   string
	password   string
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
	Pin     bool
	Tag     uint32
	BatchID string
}
