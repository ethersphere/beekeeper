package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *command) initDeleteK8SNamespace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-namespace",
		Short: "Create Kubernetes namespace",
		Long:  `Create Kubernetes namespace.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly one argument representing name of the Kubernetes namespace")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			name := args[0]

			// TODO: move to preRun
			if k8sClient == nil {
				return fmt.Errorf("k8s client not created")
			}

			if err = k8sClient.Namespace.Delete(cmd.Context(), name); err != nil {
				return fmt.Errorf("delete namespace %s: %w", name, err)
			}

			fmt.Printf("namespace %s deleted\n", name)
			return
		},
	}

	return cmd
}
