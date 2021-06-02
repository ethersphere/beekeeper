package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initCreateK8SNamespace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-namespace",
		Short: "creates Kubernetes namespace",
		Long:  `creates Kubernetes namespace.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly one argument representing name of the Kubernetes namespace")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]

			if err = c.k8sClient.Namespace.Create(cmd.Context(), name); err != nil {
				return fmt.Errorf("create namespace %s: %w", name, err)
			}

			fmt.Printf("namespace %s created\n", name)
			return
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := c.setK8S(); err != nil {
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
