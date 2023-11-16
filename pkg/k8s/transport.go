package k8s

import (
	"net/http"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// customTransport is an example custom transport that wraps the default transport
// and adds some custom behavior.
type customTransport struct {
	base        http.RoundTripper
	semaphore   chan struct{}
	rateLimiter flowcontrol.RateLimiter
	logger      logging.Logger
}

func NewCustomTransport(base http.RoundTripper, config *rest.Config, semaphore chan struct{}, logger logging.Logger) *customTransport {
	return &customTransport{
		base:        base,
		semaphore:   semaphore,
		rateLimiter: config.RateLimiter,
		logger:      logger,
	}
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Acquire the semaphore to limit the number of concurrent requests.
	t.semaphore <- struct{}{}
	defer func() {
		<-t.semaphore
	}()

	t.rateLimiter.Accept()

	// Forward the request to the base transport.
	resp, err := t.base.RoundTrip(req)

	return resp, err
}
