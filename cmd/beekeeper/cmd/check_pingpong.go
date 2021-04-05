package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pingpong"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"golang.org/x/sync/errgroup"

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
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %w", err)
			}

			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				K8SClient:           k8sClient,
				Namespace:           c.config.GetString(optionNameNamespace),
				DisableNamespace:    disableNamespace,
			})

			if startCluster {
				// bootnodes group
				bgName := "bootnodes"
				bgOptions := newDefaultNodeGroupOptions()
				bgOptions.Image = image
				bgOptions.Labels = map[string]string{
					"app.kubernetes.io/component": "bootnode",
					"app.kubernetes.io/part-of":   bgName,
					"app.kubernetes.io/version":   strings.Split(image, ":")[1],
				}
				bgOptions.PersistenceEnabled = persistence
				bgOptions.PersistenceStorageClass = storageClass
				bgOptions.PersistenceStorageRequest = storageRequest
				cluster.AddNodeGroup(bgName, *bgOptions)
				bg := cluster.NodeGroup(bgName)
				bSetup := setupBootnodes(bootnodeCount, c.config.GetString(optionNameNamespace))

				bCtx, bCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer bCancel()
				bnGroup := new(errgroup.Group)
				for i := 0; i < bootnodeCount; i++ {
					bConfig := newDefaultBeeConfig()
					bConfig.Bootnodes = bSetup[i].Bootnodes
					bName := fmt.Sprintf("bootnode-%d", i)
					bOptions := bee.NodeOptions{
						Config:       bConfig,
						ClefKey:      bSetup[i].ClefKey,
						ClefPassword: bSetup[i].ClefPassword,
						LibP2PKey:    bSetup[i].LibP2PKey,
						SwarmKey:     bSetup[i].SwarmKey,
					}

					bnGroup.Go(func() error {
						return bg.AddStartNode(bCtx, bName, bOptions)
					})
				}

				if err := bnGroup.Wait(); err != nil {
					return fmt.Errorf("starting bootnodes: %w", err)
				}
				fmt.Println("bootnodes started")

				// nodes group
				ngName := "nodes"
				ngOptions := newDefaultNodeGroupOptions()
				ngOptions.Image = image
				ngOptions.Labels = map[string]string{
					"app.kubernetes.io/component": "node",
					"app.kubernetes.io/part-of":   ngName,
					"app.kubernetes.io/version":   strings.Split(image, ":")[1],
				}
				ngOptions.PersistenceEnabled = persistence
				ngOptions.PersistenceStorageClass = storageClass
				ngOptions.PersistenceStorageRequest = storageRequest
				ngOptions.BeeConfig = newDefaultBeeConfig()
				ngOptions.BeeConfig.Bootnodes = setupBootnodesDNS(bootnodeCount, c.config.GetString(optionNameNamespace))
				cluster.AddNodeGroup(ngName, *ngOptions)
				ng := cluster.NodeGroup(ngName)

				nCtx, nCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer nCancel()
				nGroup := new(errgroup.Group)
				for i := 0; i < nodeCount; i++ {
					nName := fmt.Sprintf("bee-%d", i)

					nGroup.Go(func() error {
						return ng.AddStartNode(nCtx, nName, bee.NodeOptions{})
					})
				}

				if err := nGroup.Wait(); err != nil {
					return fmt.Errorf("starting nodes: %w", err)
				}
				fmt.Println("nodes started")

			} else {
				if bootnodeCount > 0 {
					// bootnodes group
					bgName := "bootnodes"
					bgOptions := newDefaultNodeGroupOptions()
					cluster.AddNodeGroup(bgName, *bgOptions)
					bg := cluster.NodeGroup(bgName)

					for i := 0; i < bootnodeCount; i++ {
						if err := bg.AddNode(fmt.Sprintf("bootnode-%d", i), bee.NodeOptions{}); err != nil {
							return fmt.Errorf("adding bootnode-%d: %w", i, err)
						}
					}
				}

				// nodes group
				ngName := "nodes"
				ngOptions := newDefaultNodeGroupOptions()
				ngOptions.Image = image
				ngOptions.Labels = map[string]string{
					"app.kubernetes.io/component": "node",
					"app.kubernetes.io/part-of":   ngName,
					"app.kubernetes.io/version":   strings.Split(image, ":")[1],
				}
				ngOptions.PersistenceEnabled = persistence
				ngOptions.PersistenceStorageClass = storageClass
				ngOptions.PersistenceStorageRequest = storageRequest
				ngOptions.BeeConfig = newDefaultBeeConfig()
				ngOptions.BeeConfig.Bootnodes = setupBootnodesDNS(bootnodeCount, c.config.GetString(optionNameNamespace))
				cluster.AddNodeGroup(ngName, *ngOptions)
				ng := cluster.NodeGroup(ngName)

				for i := 0; i < nodeCount; i++ {
					if err := ng.AddNode(fmt.Sprintf("bee-%d", i), bee.NodeOptions{}); err != nil {
						return fmt.Errorf("adding bee-%d: %w", i, err)
					}
				}
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
						NodeGroup:   "nodes",
						AddCount:    dynamicActions[i],
						DeleteCount: dynamicActions[i+1],
						StartCount:  dynamicActions[i+2],
						StopCount:   dynamicActions[i+3],
					})
				}

				return pingpong.CheckDynamic(cmd.Context(), cluster, pingpong.Options{
					DynamicActions: dActions,
					MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
					MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace)),
					Seed:           seed,
				})
			}

			return pingpong.Check(cmd.Context(), cluster, pingpong.Options{
				MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
				MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace)),
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
