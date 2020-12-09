package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initCreateCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create Bee",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().Bool(optionNameInCluster, false, "run Beekeeper in cluster")
	cmd.PersistentFlags().String(optionNameKubeconfig, "~/.kube/config", "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "beekeeper", "kubernetes namespace")

	cmd.AddCommand(c.initCreateNamespace())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) createPreRunE(cmd *cobra.Command, args []string) (err error) {
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}

	return
}
