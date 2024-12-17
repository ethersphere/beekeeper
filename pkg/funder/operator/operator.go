package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/node-funder/pkg/funder"
	v1 "k8s.io/api/core/v1"
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
	httpClient *http.Client
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
		httpClient:   httpClient,
		ClientConfig: cfg,
	}
}

func (c *Client) Run(ctx context.Context) error {
	c.Log.Infof("operator started for namespace %s", c.Namespace)
	defer c.Log.Info("operator done")

	newPods := make(chan *v1.Pod)
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.Log.Error("operator context canceled")
				return
			case pod, ok := <-newPods:
				if !ok {
					c.Log.Error("operator channel closed")
					return
				}

				c.Log.Debugf("operator received pod with ip: %s", pod.Status.PodIP)

				nodeInfo, _, err := c.K8sClient.Service.FindNode(ctx, c.Namespace, pod)
				if err != nil {
					c.Log.Errorf("find service for pod: %v", err)
					continue
				}

				var addresses bee.Addresses

				maxRetries := 5
				for i := 0; i < maxRetries; i++ {
					addresses, err = c.getAddresses(ctx, nodeInfo.Endpoint)
					if err != nil {
						c.Log.Errorf("get addresses (attempt %d/%d): %v", i+1, maxRetries, err)
						if i < maxRetries-1 { // Wait before retrying, except on the last attempt
							time.Sleep(1 * time.Second)
						}
						continue
					}

					c.Log.Tracef("Successfully fetched addresses on attempt %d/%d", i+1, maxRetries)
					break
				}

				if err != nil {
					c.Log.Errorf("Failed to fetch addresses after %d attempts: %v", maxRetries, err)
				}

				c.Log.Infof("node '%s' ethereum address: %s", nodeInfo.Name, addresses.Ethereum)

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

	if err := c.K8sClient.Pods.WatchNewRunning(ctx, c.Namespace, c.LabelSelector, newPods); err != nil {
		return fmt.Errorf("events watch: %w", err)
	}

	return nil
}

// getAddresses sends a request to the node to get the addresses of the node,
// which includes overlay, underlay addresses, Ethereum address, and public keys.
func (c *Client) getAddresses(ctx context.Context, endpoint string) (bee.Addresses, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/addresses", endpoint), nil)
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
