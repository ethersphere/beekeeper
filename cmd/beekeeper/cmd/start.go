package cmd

import (
	"github.com/spf13/cobra"
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

	cmd.PersistentFlags().String(optionNameAPIDomain, "staging.internal", "API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSAPI, optionNameAPIInsecureTLS, false, "skips TLS verification for API")
	cmd.PersistentFlags().String(optionNameAPIScheme, "https", "API scheme")
	cmd.PersistentFlags().String(optionNameDebugAPIDomain, "staging.internal", "debug API DNS domain")
	cmd.PersistentFlags().BoolVar(&inCluster, optionNameInCluster, false, "run Beekeeper in cluster")
	cmd.PersistentFlags().BoolVar(&insecureTLSDebugAPI, optionNameDebugAPIInsecureTLS, false, "skips TLS verification for debug API")
	cmd.PersistentFlags().String(optionNameDebugAPIScheme, "https", "debug API scheme")
	cmd.PersistentFlags().String(optionNameKubeconfig, "", "kubernetes config file")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "beekeeper", "kubernetes namespace")

	cmd.AddCommand(c.initAddStartNode())
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
