package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCreateCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates Bee infrastructure",
		Long: `Creates Bee infrastructure components in your Kubernetes cluster.

The create command provides subcommands for setting up different types of infrastructure:
• bee-cluster: Deploys a complete Bee cluster with nodes, services, and storage
• k8s-namespace: Creates a Kubernetes namespace for organizing Bee resources

Each subcommand handles the necessary Kubernetes resources, configuration, and networking
to get your Bee infrastructure running. Use --help with any subcommand for detailed options.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(c.initCreateK8sNamespace())
	cmd.AddCommand(c.initCreateBeeCluster())

	c.root.AddCommand(cmd)

	return nil
}
