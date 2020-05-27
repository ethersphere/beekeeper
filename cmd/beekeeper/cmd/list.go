package cmd

import (
	"github.com/spf13/cobra"
)

func (c *command) initListCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List info from Bee node(s)",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameAPIScheme, "https", "API scheme")
	cmd.PersistentFlags().String(optionNameAPIHostnamePattern, "bee-%d", "API hostname pattern")
	cmd.PersistentFlags().String(optionNameAPIDomain, "core.internal", "API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSAPI, optionNameAPIInsecureTLS, false, "skips TLS verification for API")
	cmd.PersistentFlags().String(optionNameDebugAPIScheme, "https", "debug API scheme")
	cmd.PersistentFlags().String(optionNameDebugAPIHostnamePattern, "bee-%d-debug", "debug API hostname pattern")
	cmd.PersistentFlags().String(optionNameDebugAPIDomain, "core.internal", "debug API DNS domain")
	cmd.PersistentFlags().BoolVar(&insecureTLSDebugAPI, optionNameDebugAPIInsecureTLS, false, "skips TLS verification for debug API")
	cmd.PersistentFlags().BoolVar(&disableNamespace, optionNameDisableNamespace, false, "disable Kubernetes namespace")
	cmd.PersistentFlags().Bool(optionNameInsecureTLS, false, "skips TLS verification for both API and debug API")
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace, must be set or disabled")
	cmd.PersistentFlags().IntP(optionNameNodeCount, "c", 1, "node count")

	cmd.AddCommand(c.initListOverlays())

	c.root.AddCommand(cmd)
	return nil
}

func (c *command) listPreRunE(cmd *cobra.Command, args []string) (err error) {
	if !disableNamespace {
		if err = cmd.MarkFlagRequired(optionNameNamespace); err != nil {
			return
		}
	}
	if err = c.config.BindPFlags(cmd.Flags()); err != nil {
		return
	}
	if !disableNamespace && len(c.config.GetString(optionNameNamespace)) == 0 {
		return cmd.Help()
	}

	if c.config.GetBool(optionNameInsecureTLS) {
		insecureTLSAPI = true
		insecureTLSDebugAPI = true
	}

	return
}
