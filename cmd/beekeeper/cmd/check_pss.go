package cmd

import (
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/check/pss"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPSS() *cobra.Command {
	const (
		optionNameSeed   = "seed"
		optionTimeout    = "timeout"
		optionAddrPrefix = "address-prefix"
	)

	cmd := &cobra.Command{
		Use:   "pss",
		Short: "Checks PSS ability of the cluster",
		Long: `Checks PSS ability of the cluster.
It establishes a WebSocket connection to random recipient nodes, and 
sends PSS messages to random nodes to check if WebSocket connections receive the data.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			return pss.Check(cluster, pss.Options{
				NodeGroup:      "bee",
				NodeCount:      c.config.GetInt(optionNameNodeCount),
				Seed:           seed,
				RequestTimeout: c.config.GetDuration(optionTimeout),
				AddressPrefix:  c.config.GetInt(optionAddrPrefix),
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for choosing random nodes; if not set, will be random")
	cmd.Flags().Duration(optionTimeout, time.Minute*5, "timeout duration for pss retrieval")
	cmd.Flags().Int(optionAddrPrefix, 1, "public address prefix bytes count")

	return cmd
}
