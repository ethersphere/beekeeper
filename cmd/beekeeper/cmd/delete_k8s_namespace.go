package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initDeleteK8SNamespace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   namespaceCmd,
		Short: "deletes Kubernetes namespace",
		Long:  `Deletes Kubernetes namespace.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly one argument representing name of the Kubernetes namespace")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]

			if err = c.k8sClient.Namespace.Delete(cmd.Context(), name); err != nil {
				return fmt.Errorf("delete namespace %s: %w", name, err)
			}

			c.log.Infof("namespace %s deleted", name)
			return err
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := c.setK8sClient(); err != nil {
				return err
			}

			if c.k8sClient == nil {
				return fmt.Errorf("k8s client not set")
			}

			return nil
		},
	}

	return cmd
}
