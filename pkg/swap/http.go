package swap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethersphere/beekeeper"
)

const contentType = "application/json; charset=utf-8"

var userAgent = "beekeeper/" + beekeeper.Version

// requestJSON handles the HTTP request response cycle. It JSON encodes the request
// body, creates an HTTP request with provided method on a path with required
// headers and decodes request body if the v argument is not nil and content type is
// application/json.
func requestJSON(ctx context.Context, httpClient *http.Client, method, path string, body, v interface{}) (err error) {
	var bodyBuffer io.ReadWriter
	if body != nil {
		bodyBuffer = new(bytes.Buffer)
		if err = encodeJSON(bodyBuffer, body); err != nil {
			return err
		}
	}

	return request(ctx, httpClient, method, path, bodyBuffer, v)
}

// request handles the HTTP request response cycle.
func request(ctx context.Context, httpClient *http.Client, method, path string, body io.Reader, v interface{}) (err error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", contentType)

	r, err := httpClient.Do(req)
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
		r.URL = baseURL
		return transport.RoundTrip(r)
	})
	return c
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

		_, _ = io.Copy(io.Discard, r)
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
