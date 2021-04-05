package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initAddStartNode() *cobra.Command {
	const (
		createdBy                  = "beekeeper"
		labelName                  = "bee"
		managedBy                  = "beekeeper"
		optionNameBootnodes        = "bootnodes"
		optionNameClusterName      = "cluster-name"
		optionNameNodeGroupName    = "node-group-name"
		optionNameNodeGroupVersion = "node-group-version"
		optionNameNodeName         = "node-name"
		optionNameStartStandalone  = "standalone"
		optionNamePersistence      = "persistence"
		optionNameStorageClass     = "storage-class"
		optionNameStorageRequest   = "storage-request"
	)

	var (
		bootnodes        string
		clusterName      string
		nodeGroupName    string
		nodeGroupVersion string
		nodeName         string
		standalone       bool
		persistence      bool
		storageClass     string
		storageRequest   string
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Start Bee node",
		Long:  `Start Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return fmt.Errorf("to be rethinked, subset of start cluster, probably not needed anymore")
		},
		PreRunE: c.startPreRunE,
	}

	cmd.PersistentFlags().StringVar(&bootnodes, optionNameBootnodes, "", "bootnodes")
	cmd.PersistentFlags().StringVar(&clusterName, optionNameClusterName, "bee", "cluster name")
	cmd.PersistentFlags().StringVar(&nodeGroupName, optionNameNodeGroupName, "bee", "node group name")
	cmd.PersistentFlags().StringVar(&nodeGroupVersion, optionNameNodeGroupVersion, "latest", "node group version")
	cmd.PersistentFlags().StringVar(&nodeName, optionNameNodeName, "bee", "node name")
	cmd.PersistentFlags().BoolVarP(&standalone, optionNameStartStandalone, "s", false, "start a standalone node")
	cmd.PersistentFlags().BoolVar(&persistence, optionNamePersistence, false, "use persistent storage")
	cmd.PersistentFlags().StringVar(&storageClass, optionNameStorageClass, "local-storage", "storage class name")
	cmd.PersistentFlags().StringVar(&storageRequest, optionNameStorageRequest, "34Gi", "storage request")

	return cmd
}
