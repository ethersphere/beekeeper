package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/spf13/cobra"
)

func (c *command) initCreateNode() *cobra.Command {
	return &cobra.Command{
		Use:   "node",
		Short: "Create Bee node",
		Long:  `Create Bee node.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientset := k8s.NewClient(&k8s.ClientOptions{KubeconfigPath: c.config.GetString(optionNameK8SConfig)})

			return k8s.Check(clientset, k8s.Options{
				Namespace: c.config.GetString(optionNameK8SNamespace),
			})
		},
		PreRunE: c.createPreRunE,
	}
}
