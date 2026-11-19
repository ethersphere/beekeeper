package httpx

import (
	"crypto/tls"
	"net/http"

	"github.com/ethersphere/beekeeper"
)

var userAgent = "beekeeper/" + beekeeper.Version

// ClientOptions configures the client returned by NewClient.
type ClientOptions struct {
	// InsecureSkipVerify disables TLS certificate verification.
	InsecureSkipVerify bool
}

// NewClient returns an HTTP client that sets the beekeeper User-Agent on every
// request. A transport pools connections per host, so a single client is meant
// to be shared by all clients that talk to Bee nodes with the same TLS
// settings, regardless of how many nodes (hosts) they address.
func NewClient(o ClientOptions) *http.Client {
	t := newTransport()
	if o.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &http.Client{
		Transport: &HeaderRoundTripper{Next: t},
	}
}

// newTransport returns a copy of http.DefaultTransport, keeping its proxy,
// timeout, connection pool and HTTP/2 defaults.
func newTransport() *http.Transport {
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		return t.Clone()
	}

	// http.DefaultTransport was replaced with a different implementation.
	return &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,
	}
}

// HeaderRoundTripper sets the beekeeper User-Agent header on outgoing requests
// and delegates them to Next, or to http.DefaultTransport if Next is nil.
type HeaderRoundTripper struct {
	Next http.RoundTripper
}

func (hrt *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// A RoundTripper must not modify the request it was given.
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", userAgent)

	return hrt.next().RoundTrip(req)
}

// CloseIdleConnections closes idle connections of Next, so that
// http.Client.CloseIdleConnections reaches the wrapped transport.
func (hrt *HeaderRoundTripper) CloseIdleConnections() {
	type closeIdler interface{ CloseIdleConnections() }

	if c, ok := hrt.next().(closeIdler); ok {
		c.CloseIdleConnections()
	}
}

func (hrt *HeaderRoundTripper) next() http.RoundTripper {
	if hrt.Next == nil {
		return http.DefaultTransport
	}
	return hrt.Next
}
