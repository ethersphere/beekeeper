package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPingPong() *cobra.Command {
	const (
		optionNameStartCluster   = "start-cluster"
		optionNameDynamic        = "dynamic"
		optionNameDynamicActions = "dynamic-actions"
		optionNameClusterName    = "cluster-name"
		optionNameBootnodeCount  = "bootnode-count"
		optionNameNodeCount      = "node-count"
		optionNameImage          = "bee-image"
		optionNameSeed           = "seed"
		optionNamePersistence    = "persistence"
		optionNameStorageClass   = "storage-class"
		optionNameStorageRequest = "storage-request"
	)

	var (
		startCluster   bool
		dynamic        bool
		dynamicActions []int
		clusterName    string
		bootnodeCount  int
		nodeCount      int
		image          string
		persistence    bool
		storageClass   string
		storageRequest string
	)

	cmd := &cobra.Command{
		Use:   "pingpong",
		Short: "Executes ping from all nodes to all other nodes in the cluster",
		Long: `Executes ping from all nodes to all other nodes in the cluster,
and prints round-trip time (RTT) of each ping.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")

			cluster, err := setupCluster(cmd.Context(), cfg, startCluster)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			if dynamic {
				if len(dynamicActions)%4 != 0 {
					return fmt.Errorf("number of dynamic actions must be divisable by 4")
				}
				dActions := []pingpong.Actions{}
				for i := 0; i < len(dynamicActions); i = i + 4 {
					dActions = append(dActions, pingpong.Actions{
						NodeGroup:   "bee",
						AddCount:    dynamicActions[i],
						DeleteCount: dynamicActions[i+1],
						StartCount:  dynamicActions[i+2],
						StopCount:   dynamicActions[i+3],
					})
				}

				return pingpong.CheckDynamic(cmd.Context(), cluster, pingpong.Options{
					DynamicActions: dActions,
					MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
					MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), cfg.Cluster.Namespace),
					Seed:           seed,
				})
			}

			return pingpong.Check(cmd.Context(), cluster, pingpong.Options{
				MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
				MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), cfg.Cluster.Namespace),
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")
	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "check on dynamic cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.Flags().IntSliceVar(&dynamicActions, optionNameDynamicActions, []int{1, 1, 0, 1, 2, 1, 1, 2}, "passed in groups of 4 dynamic actions: add, delete, start, stop")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}
