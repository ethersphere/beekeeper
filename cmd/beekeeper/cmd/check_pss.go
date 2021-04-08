package cmd

import (
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pss"
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

			cluster := bee.NewCluster("bee", bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			ngOptions := newDefaultNodeGroupOptions()
			cluster.AddNodeGroup("nodes", *ngOptions)
			ng := cluster.NodeGroup("nodes")

			for i := 0; i < c.config.GetInt(optionNameNodeCount); i++ {
				if err := ng.AddNode(fmt.Sprintf("bee-%d", i), bee.NodeOptions{}); err != nil {
					return fmt.Errorf("adding node bee-%d: %s", i, err)
				}
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			return pss.Check(cluster, pss.Options{
				NodeGroup:      "nodes",
				NodeCount:      c.config.GetInt(optionNameNodeCount),
				Seed:           seed,
				RequestTimeout: c.config.GetDuration(optionTimeout),
				AddressPrefix:  c.config.GetInt(optionAddrPrefix),
				PostageAmount:  c.config.GetInt64(optionNamePostageAmount),
				PostageWait:    c.config.GetDuration(optionNamePostageBatchhWait),
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for choosing random nodes; if not set, will be random")
	cmd.Flags().Duration(optionTimeout, time.Minute*5, "timeout duration for pss retrieval")
	cmd.Flags().Int(optionAddrPrefix, 1, "public address prefix bytes count")

	return cmd
}
