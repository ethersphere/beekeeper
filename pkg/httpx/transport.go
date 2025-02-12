package httpx

import (
	"net/http"

	"github.com/ethersphere/beekeeper"
)

var userAgent = "beekeeper/" + beekeeper.Version

type HeaderRoundTripper struct {
	Next http.RoundTripper
}

func (hrt *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	return hrt.Next.RoundTrip(req)
}
