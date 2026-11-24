package httpx

import (
	"crypto/tls"
	"net/http"

	"github.com/ethersphere/beekeeper"
)

var userAgent = "beekeeper/" + beekeeper.Version

// ClientOptions configures NewClient.
type ClientOptions struct {
	InsecureSkipVerify bool
}

// NewClient returns a client that sets the beekeeper User-Agent. Share one
// client across nodes; its transport pools connections per host.
func NewClient(o ClientOptions) *http.Client {
	t := newTransport()
	if o.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &http.Client{
		Transport: &HeaderRoundTripper{Next: t},
	}
}

// newTransport clones http.DefaultTransport to keep its timeout and pool defaults.
func newTransport() *http.Transport {
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		return t.Clone()
	}

	return &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,
	}
}

// HeaderRoundTripper sets the beekeeper User-Agent header. A nil Next means
// http.DefaultTransport.
type HeaderRoundTripper struct {
	Next http.RoundTripper
}

func (hrt *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// RoundTrip must not modify the caller's request.
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", userAgent)

	return hrt.next().RoundTrip(req)
}

// CloseIdleConnections lets http.Client.CloseIdleConnections reach Next.
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
