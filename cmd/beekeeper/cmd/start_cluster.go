package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

const (
	// CICD options
	optionNameClefSignerEnable   = "clef-signer-enable"
	optionNameDBCapacity         = "db-capacity"
	optionNamePaymentEarly       = "payment-early"
	optionNamePaymentThreshold   = "payment-threshold"
	optionNamePaymentTolerance   = "payment-tolerance"
	optionNameSwapEnable         = "swap-enable"
	optionNameSwapEndpoint       = "swap-endpoint"
	optionNameSwapFactoryAddress = "swap-factory-address"
	optionNameSwapInitialDeposit = "swap-initial-deposit"
	optionNameNodeSelector       = "node-selector"
	optionNameIngressClass       = "ingress-class"
)

var (
	// CICD options
	clefSignerEnable   bool
	dbCapacity         uint64
	paymentEarly       uint64
	paymentThreshold   uint64
	paymentTolerance   uint64
	swapEnable         bool
	swapEndpoint       string
	swapFactoryAddress string
	swapInitialDeposit uint64
	nodeSelector       string
	ingressClass       string
)

func (c *command) initStartCluster() *cobra.Command {
	const (
		createdBy                          = "beekeeper"
		labelName                          = "bee"
		managedBy                          = "beekeeper"
		optionNameClusterName              = "cluster-name"
		optionNameImagePullSecrets         = "image-pull-secrets"
		optionNameBootnodeCount            = "bootnode-count"
		optionNameNodeCount                = "node-count"
		optionNameImage                    = "bee-image"
		optionNameFullNode                 = "full-node"
		optionNamePersistence              = "persistence"
		optionNameStorageClass             = "storage-class"
		optionNameStorageRequest           = "storage-request"
		optionNameAdditionalNodeCount      = "additional-node-count"
		optionNameAdditionalImage          = "additional-bee-image"
		optionNameAdditionalFullNode       = "additional-full-node"
		optionNameAdditionalPersistence    = "additional-persistence"
		optionNameAdditionalStorageClass   = "additional-storage-class"
		optionNameAdditionalStorageRequest = "additional-storage-request"
	)

	var (
		clusterName              string
		imagePullSecrets         []string
		bootnodeCount            int
		nodeCount                int
		image                    string
		fullNode                 bool
		persistence              bool
		storageClass             string
		storageRequest           string
		additionalNodeCount      int
		additionalImage          string
		additionalFullNode       bool
		additionalPersistence    bool
		additionalStorageClass   string
		additionalStorageRequest string
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Start Bee cluster",
		Long:  `Start Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			k8sClient, err := setK8SClient(c.config.GetString(optionNameKubeconfig), c.config.GetBool(optionNameInCluster))
			if err != nil {
				return fmt.Errorf("creating Kubernetes client: %w", err)
			}

			namespace := c.config.GetString(optionNameNamespace)
			cluster := bee.NewCluster(clusterName, bee.ClusterOptions{
				Annotations: map[string]string{
					"created-by":        createdBy,
					"beekeeper/version": beekeeper.Version,
				},
				APIDomain:           c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:      insecureTLSAPI,
				APIScheme:           c.config.GetString(optionNameAPIScheme),
				DebugAPIDomain:      c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS: insecureTLSDebugAPI,
				DebugAPIScheme:      c.config.GetString(optionNameDebugAPIScheme),
				K8SClient:           k8sClient,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": managedBy,
					"app.kubernetes.io/name":       labelName,
				},
				Namespace: namespace,
			})

			cicd := newCICDOptions(clefSignerEnable, dbCapacity, paymentEarly, paymentThreshold, paymentTolerance, swapEnable, swapEndpoint, swapFactoryAddress, swapInitialDeposit, nodeSelector, ingressClass)

			// bootnodes group
			bgName := "bootnode"
			bCtx, bCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer bCancel()
			if err := startBootNodeGroup(bCtx, cluster, bootnodeCount, nodeCount, bgName, namespace, image, storageClass, storageRequest, imagePullSecrets, persistence, cicd); err != nil {
				return fmt.Errorf("starting bootnode group %s: %w", bgName, err)
			}

			// node groups
			ngName := "bee"
			nCtx, nCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer nCancel()
			if err := startNodeGroup(nCtx, cluster, bootnodeCount, nodeCount, ngName, namespace, image, storageClass, storageRequest, imagePullSecrets, persistence, fullNode, cicd); err != nil {
				return fmt.Errorf("starting node group %s: %w", ngName, err)
			}

			if additionalNodeCount > 0 {
				addNgName := "drone"
				addNCtx, addNCancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
				defer addNCancel()
				if err := startNodeGroup(addNCtx, cluster, bootnodeCount, additionalNodeCount, addNgName, namespace, additionalImage, additionalStorageClass, additionalStorageRequest, imagePullSecrets, additionalPersistence, additionalFullNode, cicd); err != nil {
					return fmt.Errorf("starting node group %s: %w", addNgName, err)
				}
			}

			return
		},
		PreRunE: c.startPreRunE,
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().StringArrayVar(&imagePullSecrets, optionNameImagePullSecrets, []string{"regcred"}, "image pull secrets")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 1, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.PersistentFlags().BoolVar(&fullNode, optionNameFullNode, true, "start node in full mode")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, true, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "ethersphere/bee:latest", "Bee Docker image in additional node group")
	cmd.PersistentFlags().BoolVar(&additionalFullNode, optionNameAdditionalFullNode, false, "start node in full mode")
	cmd.PersistentFlags().BoolVar(&additionalPersistence, optionNameAdditionalPersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&additionalStorageClass, optionNameAdditionalStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&additionalStorageRequest, optionNameAdditionalStorageRequest, "34Gi", "storage request")
	// CICD options
	cmd.Flags().BoolVar(&clefSignerEnable, optionNameClefSignerEnable, false, "enable Clef signer")
	cmd.Flags().Uint64Var(&dbCapacity, optionNameDBCapacity, 5000000, "DB capacity")
	cmd.Flags().Uint64Var(&paymentEarly, optionNamePaymentEarly, 100000000000, "payment early")
	cmd.Flags().Uint64Var(&paymentThreshold, optionNamePaymentThreshold, 1000000000000, "payment threshold")
	cmd.Flags().Uint64Var(&paymentTolerance, optionNamePaymentTolerance, 100000000000, "payment tolerance")
	cmd.Flags().BoolVar(&swapEnable, optionNameSwapEnable, false, "enable swap")
	cmd.Flags().StringVar(&swapEndpoint, optionNameSwapEndpoint, "ws://geth-swap.geth:8546", "swap endpoint")
	cmd.Flags().StringVar(&swapFactoryAddress, optionNameSwapFactoryAddress, "0x657241f4494a2f15ba75346e691d753a978c72df", "swap factory address")
	cmd.Flags().Uint64Var(&swapInitialDeposit, optionNameSwapInitialDeposit, 500000000000000000, "swap initial deposit")
	cmd.Flags().StringVar(&nodeSelector, optionNameNodeSelector, "bee-staging", "node selector")
	cmd.Flags().StringVar(&ingressClass, optionNameIngressClass, "nginx-internal", "ingress class")

	return cmd
}
