package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initStartCluster() *cobra.Command {
	const (
		createdBy                     = "beekeeper"
		labelName                     = "bee"
		managedBy                     = "beekeeper"
		optionNameClusterName         = "cluster-name"
		optionNameImage               = "bee-image"
		optionNameAdditionalImage     = "additional-bee-image"
		optionNameBootnodeCount       = "bootnode-count"
		optionNameNodeCount           = "node-count"
		optionNameAdditionalNodeCount = "additional-node-count"
		optionNamePersistence         = "persistence"
		optionNameStorageClass        = "storage-class"
		optionNameStorageRequest      = "storage-request"
	)

	var (
		clusterName         string
		image               string
		additionalImage     string
		bootnodeCount       int
		nodeCount           int
		additionalNodeCount int
		persistence         bool
		storageClass        string
		storageRequest      string
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Start Bee cluster",
		Long:  `Start Bee cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := config.Read("config.yaml")
			if err != nil {
				return err
			}
			_, err = setupCluster(cmd.Context(), cfg, true)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			return
		},
		PreRunE: c.startPreRunE,
	}

	cmd.Flags().StringVar(&clusterName, optionNameClusterName, "beekeeper", "cluster name")
	cmd.Flags().StringVar(&image, optionNameImage, "ethersphere/bee:latest", "Bee Docker image")
	cmd.Flags().StringVar(&additionalImage, optionNameAdditionalImage, "ethersphere/bee-netem:latest", "Bee Docker image in additional node group")
	cmd.Flags().IntVarP(&bootnodeCount, optionNameBootnodeCount, "b", 1, "number of bootnodes")
	cmd.Flags().IntVarP(&nodeCount, optionNameNodeCount, "c", 1, "number of nodes")
	cmd.Flags().IntVar(&additionalNodeCount, optionNameAdditionalNodeCount, 0, "number of nodes in additional node group")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}
