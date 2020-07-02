package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/helm3"
	"github.com/spf13/cobra"
)

func (c *command) initHelmUpgrade() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "helm upgrade",
		Long:  `helm upgrade.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			kubeconfig := c.config.GetString(optionNameHelmConfig)
			namespace := c.config.GetString(optionNameHelmNamespace)
			release := c.config.GetString(optionNameRelease)
			chart := c.config.GetString(optionNameChart)
			vals := c.config.GetString(optionNameArgs)

			values := map[string]string{
				"set": vals,
			}

			err = helm3.Upgrade(kubeconfig, namespace, release, chart, values)
			if err != nil {
				return err
			}
			return
		},
		PreRunE: c.helmPreRunE,
	}
}
