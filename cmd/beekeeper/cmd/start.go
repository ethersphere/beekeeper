package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameStartKubeconfig = "kubeconfig"
	optionNameStartNamespace  = "namespace"
)

func (c *command) initStartCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start Bee",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameStartKubeconfig, "~/.kube/config", "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameStartNamespace, "n", "beekeeper", "kubernetes namespace")

	cmd.AddCommand(c.initStartNode())
	cmd.AddCommand(c.initStartCluster())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) startPreRunE(cmd *cobra.Command, args []string) (err error) {
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}

	return
}
