package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameK8SConfig    = "kubeconfig"
	optionNameK8SNamespace = "namespace"
)

func (c *command) initK8SCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameK8SConfig, "~/.kube/config", "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameK8SNamespace, "n", "beekeeper", "kubernetes namespace")

	cmd.AddCommand(c.initK8SCheck())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) k8sPreRunE(cmd *cobra.Command, args []string) (err error) {
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}

	return
}
