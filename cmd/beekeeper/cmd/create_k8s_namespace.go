package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const namespaceCmd string = "k8s-namespace"

func (c *command) initCreateK8sNamespace() *cobra.Command {
	cmd := &cobra.Command{
		Use:   namespaceCmd,
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

			if _, err = c.k8sClient.Namespace.Create(cmd.Context(), name); err != nil {
				return fmt.Errorf("create namespace %s: %w", name, err)
			}

			c.log.Infof("namespace %s created", name)
			return
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
