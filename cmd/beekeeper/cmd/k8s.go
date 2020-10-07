package cmd

import (
	"github.com/spf13/cobra"
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

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) k8sPreRunE(cmd *cobra.Command, args []string) (err error) {
	// if !disableNamespace && len(c.config.GetString(optionNameNamespace)) == 0 {
	// 	if err = cmd.MarkFlagRequired(optionNameNamespace); err != nil {
	// 		return
	// 	}
	// }
	// if err = c.config.BindPFlags(cmd.Flags()); err != nil {
	// 	return
	// }
	// if !disableNamespace && len(c.config.GetString(optionNameNamespace)) == 0 {
	// 	return cmd.Help()
	// }

	// if c.config.GetBool(optionNameInsecureTLS) {
	// 	insecureTLSAPI = true
	// 	insecureTLSDebugAPI = true
	// }

	return
}
