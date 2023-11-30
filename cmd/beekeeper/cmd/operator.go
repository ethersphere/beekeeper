package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/node-funder/pkg/funder"
	"github.com/spf13/cobra"
)

func (c *command) initOperatorCmd() (err error) {
	const (
		optionNameNamespace         = "namespace"
		optionNameChainNodeEndpoint = "geth-url"
		optionNameWalletKey         = "wallet-key"
		optionNameMinNative         = "min-native"
		optionNameMinSwarm          = "min-swarm"
		optionNameTimeout           = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "scans for scheduled pods and funds them",
		Long:  `Operator scans for scheduled pods and funds them using node-funder. beekeeper operator`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.NodeFunder{}
			namespace := c.globalConfig.GetString(optionNameNamespace)

			// chain node endpoint check
			if cfg.ChainNodeEndpoint = c.globalConfig.GetString(optionNameChainNodeEndpoint); cfg.ChainNodeEndpoint == "" {
				return errors.New("chain node endpoint (geth-url) not provided")
			}

			// wallet key check
			if cfg.WalletKey = c.globalConfig.GetString(optionNameWalletKey); cfg.WalletKey == "" {
				return errors.New("wallet key not provided")
			}

			cfg.MinAmounts.NativeCoin = c.globalConfig.GetFloat64(optionNameMinNative)
			cfg.MinAmounts.SwarmToken = c.globalConfig.GetFloat64(optionNameMinSwarm)

			// add timeout to operator
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			c.log.Infof("operator started")
			defer c.log.Infof("operator done")

			operatorChan := make(chan string)
			go func() {
				client := &http.Client{}

				for {
					select {
					case <-ctx.Done():
						c.log.Error("operator context canceled")
						return
					case podIp, ok := <-operatorChan:
						if !ok {
							c.log.Error("operator channel closed")
							return
						}
						c.log.Debugf("operator received pod ip: %s", podIp)

						// curl http://10.3.247.202:1635/addresses
						// get ehreum address from this url
						// bee.Addresses is struct that represents response with field Ethereum string
						url := &url.URL{
							Scheme: "http",
							Host:   podIp + ":1635",
							Path:   "/addresses",
						}

						req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
						if err != nil {
							c.log.Errorf("new request: %s", err.Error())
							continue
						}

						resp, err := client.Do(req)
						if err != nil {
							c.log.Errorf("do request: %s", err.Error())
							continue
						}

						// Read and print the response body
						body, err := io.ReadAll(resp.Body)
						if err != nil {
							// Handle error
							c.log.Errorf("read body: %s", err.Error())
							continue
						}

						c.log.Debugf("response body: %s", string(body))

						// decode response body
						var addresses bee.Addresses
						err = json.Unmarshal(body, &addresses)
						if err != nil {
							c.log.Errorf("unmarshal body: %s", err.Error())
							continue
						}

						c.log.Infof("ethereum address: %s", addresses.Ethereum)

						funder.Fund(ctx, funder.Config{
							Addresses:         []string{addresses.Ethereum},
							ChainNodeEndpoint: cfg.ChainNodeEndpoint,
							WalletKey:         cfg.WalletKey,
							MinAmounts: funder.MinAmounts{
								NativeCoin: cfg.MinAmounts.NativeCoin,
								SwarmToken: cfg.MinAmounts.SwarmToken,
							},
						}, nil, nil)

						// Somethimes we have two Running events for one pod.
						// For example, a readiness probe might succeed, triggering an update, or a container might restart within the pod.
					}
				}
			}()
			err = c.k8sClient.Pods.EventsWatch(ctx, namespace, operatorChan)
			if err != nil {
				return fmt.Errorf("events watch: %v", err)
			}
			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace to scan for scheduled pods.")
	cmd.Flags().String(optionNameChainNodeEndpoint, "", "Endpoint to chain node. Required.")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Float64(optionNameMinNative, 0, "Minimum amount of chain native coins (xDAI) nodes should have.")
	cmd.Flags().Float64(optionNameMinSwarm, 0, "Minimum amount of swarm tokens (xBZZ) nodes should have.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Timeout.")

	c.root.AddCommand(cmd)

	return nil
}
