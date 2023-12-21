package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/node-funder/pkg/funder"
)

type ClientConfig struct {
	Log               logging.Logger
	Namespace         string
	WalletKey         string
	ChainNodeEndpoint string
	MinAmounts        config.MinAmounts
	K8sClient         *k8s.Client
	HTTPClient        *http.Client // injected HTTP client
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

	operatorChan := make(chan string)
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.Log.Error("operator context canceled")
				return
			case podIp, ok := <-operatorChan:
				if !ok {
					c.Log.Error("operator channel closed")
					return
				}
				c.Log.Debugf("operator received pod ip: %s", podIp)

				addresses, err := c.processPodIP(ctx, podIp)
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
						NativeCoin: c.MinAmounts.NativeCoin,
						SwarmToken: c.MinAmounts.SwarmToken,
					},
				}, nil, nil, funder.WithLoggerOption(c.Log))
				if err != nil {
					c.Log.Errorf("funder: %v", err)
					continue
				}
			}
		}
	}()

	err := c.K8sClient.Pods.EventsWatch(ctx, c.Namespace, operatorChan)
	if err != nil {
		return fmt.Errorf("events watch: %v", err)
	}
	return nil
}

func (c *Client) processPodIP(ctx context.Context, podIp string) (bee.Addresses, error) {
	// http://10.3.247.202:1635/addresses
	// bee.Addresses is struct that represents response with field Ethereum string
	url := &url.URL{
		Scheme: "http",
		Host:   podIp + ":1635", // it is possible to extract debug port from service
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return bee.Addresses{}, fmt.Errorf("read body: %s", err.Error())
	}

	var addresses bee.Addresses
	err = json.Unmarshal(body, &addresses)
	if err != nil {
		return bee.Addresses{}, fmt.Errorf("unmarshal body: %s", err.Error())
	}

	return addresses, nil
}
