package stamper

import (
	"io"
	"net/http"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

type ClientConfig struct {
	Log           logging.Logger
	Namespace     string
	K8sClient     *k8s.Client
	HTTPClient    *http.Client // injected HTTP client
	LabelSelector string
}

type Client struct {
	*ClientConfig
	httpClient http.Client
}

func NewClient(cfg *ClientConfig) *Client {
	if cfg == nil {
		return nil
	}

	if cfg.Log == nil {
		cfg.Log = logging.New(io.Discard, 0)
	}

	// use the injected HTTP client if available, else create a new one
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		httpClient:   *httpClient,
		ClientConfig: cfg,
	}
}
