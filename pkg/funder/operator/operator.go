package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/node-funder/pkg/funder"
)

type ClientConfig struct {
	Log               logging.Logger
	Namespace         string
	WalletKey         string
	ChainNodeEndpoint string
	NativeToken       float64
	SwarmToken        float64
	K8sClient         *k8s.Client
	HTTPClient        *http.Client
	LabelSelector     string
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

func (c *Client) Run(ctx context.Context) error {
	c.Log.Infof("operator started")
	defer c.Log.Infof("operator done")

	newPodIps := make(chan string)
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.Log.Error("operator context canceled")
				return
			case podIp, ok := <-newPodIps:
				if !ok {
					c.Log.Error("operator channel closed")
					return
				}
				c.Log.Debugf("operator received pod ip: %s", podIp)

				addresses, err := c.getAddresses(ctx, podIp)
				if err != nil {
					c.Log.Errorf("process pod ip: %v", err)
					continue
				}

				c.Log.Infof("ethereum address: %s", addresses.Ethereum)

				err = funder.Fund(ctx, funder.Config{
					Addresses:         []string{addresses.Ethereum},
					ChainNodeEndpoint: c.ChainNodeEndpoint,
					WalletKey:         c.WalletKey,
					MinAmounts: funder.MinAmounts{
						NativeCoin: c.NativeToken,
						SwarmToken: c.SwarmToken,
					},
				}, nil, nil, funder.WithLoggerOption(c.Log))
				if err != nil {
					c.Log.Errorf("funder: %v", err)
				}
			}
		}
	}()

	if err := c.K8sClient.Pods.WatchNewRunning(ctx, c.Namespace, c.LabelSelector, newPodIps); err != nil {
		return fmt.Errorf("events watch: %w", err)
	}

	return nil
}

// getAddresses sends a request to the pod IP and retrieves the Addresses struct,
// which includes overlay, underlay addresses, Ethereum address, and public keys.
func (c *Client) getAddresses(ctx context.Context, podIp string) (bee.Addresses, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   podIp + ":1633", // it is possible to extract port from service
		Path:   "/addresses",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return bee.Addresses{}, fmt.Errorf("new request: %s", err.Error())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return bee.Addresses{}, fmt.Errorf("do request: %s", err.Error())
	}
	defer resp.Body.Close()

	var addresses bee.Addresses

	if err = json.NewDecoder(resp.Body).Decode(&addresses); err != nil {
		return bee.Addresses{}, fmt.Errorf("decode body: %s", err.Error())
	}

	return addresses, nil
}
