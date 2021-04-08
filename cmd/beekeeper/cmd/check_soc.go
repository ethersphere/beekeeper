package cmd

import (
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/soc"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckSOC() *cobra.Command {
	const (
		optionNameSeed = "seed"
		optionTimeout  = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "soc",
		Short: "Checks SOC ability of the cluster",
		Long:  `Checks SOC ability of the cluster. First a SOC is uploaded and then retrieved using the returned reference`,
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

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			return soc.Check(cluster, soc.Options{
				NodeGroup:      "nodes",
				PostageAmount:  c.config.GetInt64(optionNamePostageAmount),
				PostageWait:    c.config.GetDuration(optionNamePostageBatchhWait),
				RequestTimeout: c.config.GetDuration(optionTimeout),
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for choosing random nodes; if not set, will be random")
	cmd.Flags().Duration(optionTimeout, time.Minute*5, "timeout duration for soc check")

	return cmd
}
