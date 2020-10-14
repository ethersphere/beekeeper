package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/namespace"
	"github.com/spf13/cobra"
)

func (c *command) initCreateNamespace() *cobra.Command {
	return &cobra.Command{
		Use:   "namespace",
		Short: "Create Kubernetes namespace",
		Long:  `Create Kubernetes namespace.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			kubeconfig := c.config.GetString(optionNameK8SConfig)
			ns := c.config.GetString(optionNameK8SNamespace)

			k := k8s.NewClient(&k8s.ClientOptions{
				KubeconfigPath: kubeconfig,
			})

			ctx := context.Background()
			if err = k.Namespace.Create(ctx, ns, namespace.Options{
				Annotations: map[string]string{
					"createdBy": "beekeeper",
				},
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "beekeeper",
					"beekeeper/version":            beekeeper.Version,
				},
			}); err != nil {
				return
			}

			fmt.Printf("namespace %s created\n", ns)
			return
		},
		PreRunE: c.createPreRunE,
	}
}
