package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameK8SConfig    = "kubeconfig"
	optionNameK8SNamespace = "namespace"
)

func (c *command) initCreateCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create Bee",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameK8SConfig, "~/.kube/config", "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameK8SNamespace, "n", "beekeeper", "kubernetes namespace")

	cmd.AddCommand(c.initCreateNode())
	cmd.AddCommand(c.initCreateCluster())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) createPreRunE(cmd *cobra.Command, args []string) (err error) {
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}

	return
}
