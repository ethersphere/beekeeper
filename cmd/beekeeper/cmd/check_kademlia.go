package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/kademlia"
	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
)

func (c *command) initCheckKademlia() *cobra.Command {
	const (
		optionNameStartCluster   = "start-cluster"
		optionNameDynamic        = "dynamic"
		optionNameClusterName    = "cluster-name"
		optionNameBootnodeCount  = "bootnode-count"
		optionNameNodeCount      = "node-count"
		optionNameImage          = "bee-image"
		optionNamePersistence    = "persistence"
		optionNameStorageClass   = "storage-class"
		optionNameStorageRequest = "storage-request"
	)

	var (
		startCluster   bool
		dynamic        bool
		clusterName    string
		bootnodeCount  int
		nodeCount      int
		image          string
		persistence    bool
		storageClass   string
		storageRequest string
	)

	cmd := &cobra.Command{
		Use:   "kademlia",
		Short: "Checks Kademlia topology in the cluster",
		Long:  `Checks Kademlia topology in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %v", err)
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
				bgOptions.PersistanceStorageRequest = storageRequest
				cluster.AddNodeGroup(bgName, *bgOptions)
				bg := cluster.NodeGroup(bgName)
				bSetup := setupBootnodes(bootnodeCount, c.config.GetString(optionNameNamespace))

				ctxBN, cancelBN := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer cancelBN()
				errGroupBN := new(errgroup.Group)
				for i := 0; i < bootnodeCount; i++ {
					bConfig := newBeeDefaultConfig()
					bConfig.Bootnodes = bSetup[i].Bootnodes
					wbn, err := bg.AddStartNode(cmd.Context(), fmt.Sprintf("bootnode-%d", i), bee.StartNodeOptions{
						Config:       *bConfig,
						ClefKey:      bSetup[i].ClefKey,
						ClefPassword: bSetup[i].ClefPassword,
						LibP2PKey:    bSetup[i].LibP2PKey,
						SwarmKey:     bSetup[i].SwarmKey,
					})
					if err != nil {
						return fmt.Errorf("starting bootnode-%d: %s", i, err)
					}

					errGroupBN.Go(func() error {
						return wbn(ctxBN)
					})
				}

				if err := errGroupBN.Wait(); err == nil {
					fmt.Println("bootnodes started")
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
				ngOptions.PersistanceStorageRequest = storageRequest
				cluster.AddNodeGroup(ngName, *ngOptions)
				ng := cluster.NodeGroup(ngName)
				nConfig := newBeeDefaultConfig()
				nConfig.Bootnodes = setupBootnodesDNS(bootnodeCount, c.config.GetString(optionNameNamespace))

				ctxN, cancelN := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer cancelN()
				errGroupN := new(errgroup.Group)
				for i := 0; i < nodeCount; i++ {
					wn, err := ng.AddStartNode(cmd.Context(), fmt.Sprintf("bee-%d", i), bee.StartNodeOptions{
						Config: *nConfig,
					})
					if err != nil {
						return fmt.Errorf("starting bee-%d: %s", i, err)
					}

					errGroupN.Go(func() error {
						return wn(ctxN)
					})
				}

				if err := errGroupN.Wait(); err == nil {
					fmt.Println("nodes started")
				}

			} else {
				if bootnodeCount > 0 {
					// bootnodes group
					bgName := "bootnodes"
					bgOptions := newDefaultNodeGroupOptions()
					cluster.AddNodeGroup(bgName, *bgOptions)
					bg := cluster.NodeGroup(bgName)

					for i := 0; i < bootnodeCount; i++ {
						if err := bg.AddNode(fmt.Sprintf("bootnode-%d", i)); err != nil {
							return fmt.Errorf("adding bootnode-%d: %v", i, err)
						}
					}
				}

				// nodes group
				ngName := "nodes"
				ngOptions := newDefaultNodeGroupOptions()
				cluster.AddNodeGroup(ngName, *ngOptions)
				ng := cluster.NodeGroup(ngName)

				for i := 0; i < nodeCount; i++ {
					if err := ng.AddNode(fmt.Sprintf("bee-%d", i)); err != nil {
						return fmt.Errorf("adding bee-%d: %v", i, err)
					}
				}
			}

			if dynamic {
				return kademlia.CheckDynamic(cmd.Context(), cluster, []kademlia.DynamicActions{
					{
						NodeGroup:   "nodes",
						AddCount:    1,
						StartCount:  1,
						StopCount:   1,
						DeleteCount: 1,
					},
					{
						NodeGroup:   "nodes",
						AddCount:    2,
						StartCount:  2,
						StopCount:   2,
						DeleteCount: 2,
					},
					{
						NodeGroup:   "nodes",
						AddCount:    3,
						StartCount:  3,
						StopCount:   3,
						DeleteCount: 3,
					},
				})
			}

			return kademlia.Check(cmd.Context(), cluster)
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")
	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "check on dynamic cluster")
	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 0, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}
