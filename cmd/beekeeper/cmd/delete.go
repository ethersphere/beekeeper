package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initDeleteCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes Bee infrastructure",
		Long: `Deletes Bee infrastructure components from your Kubernetes cluster.

The delete command provides subcommands for removing different types of infrastructure:
• bee-cluster: Removes the entire Bee cluster including nodes, services, and storage
• k8s-namespace: Deletes a Kubernetes namespace and all its resources

Use --with-storage flag when deleting clusters to also remove persistent storage.
This command is useful for cleanup, testing, or when you need to recreate infrastructure.
Use --help with any subcommand for detailed options.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
	}

	cmd.AddCommand(c.initDeleteK8SNamespace())
	cmd.AddCommand(c.initDeleteBeeCluster())

	c.root.AddCommand(cmd)

	return nil
}
