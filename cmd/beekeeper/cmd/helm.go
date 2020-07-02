package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	optionNameHelmConfig    = "kubeconfig"
	optionNameHelmNamespace = "namespace"
	optionNameRelease       = "release"
	optionNameChart         = "chart"
	optionNameArgs          = "args"
)

func (c *command) initHelmCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "helm",
		Short: "Execute helm command on a Bee cluster",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}
	viper.AutomaticEnv()
	cmd.PersistentFlags().String(optionNameHelmConfig, viper.GetString("KUBECONFIG"), "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameHelmNamespace, "n", "bee", "kubernetes namespace")
	cmd.PersistentFlags().String(optionNameRelease, "bee", "defines the mode to run chaos scenario [one|all|fixed|fixed-percent|random-max-percent]")
	cmd.PersistentFlags().String(optionNameChart, "ethersphere/bee", "depends on the mode, for one and all leave empty")
	cmd.PersistentFlags().String(optionNameArgs, "", "if not specified it will use random bee pod from the namespace")

	cmd.AddCommand(c.initHelmUpgrade())

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) helmPreRunE(cmd *cobra.Command, args []string) (err error) {
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}

	return
}
