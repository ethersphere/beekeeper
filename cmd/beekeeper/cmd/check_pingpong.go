package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPingPong() *cobra.Command {
	const (
		optionNameNodeCount = "node-count"
	)

	var (
		nodeCount int
	)

	cmd := &cobra.Command{
		Use:   "pingpong",
		Short: "Executes ping from all nodes to all other nodes in the cluster",
		Long: `Executes ping from all nodes to all other nodes in the cluster,
and prints round-trip time (RTT) of each ping.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")

			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			return pingpong.Check(cmd.Context(), cluster, pingpong.Options{
				MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
				MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace)),
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")

	return cmd
}
