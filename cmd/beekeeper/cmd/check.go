package cmd

import (
	"github.com/spf13/cobra"
)

const (
	optionNameDebugAPIURLTemplate = "debug-api-url-template"
	optionNameNamespace           = "namespace"
	optionNameNodeCount           = "node-count"
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run tests on Bee node(s)",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flag(optionNameDebugAPIURLTemplate).Value.String() == "" {
				if err := cmd.MarkFlagRequired(optionNameNamespace); err != nil {
					panic(err)
				}
			}
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().String(optionNameDebugAPIURLTemplate, "", "Debug API URL template")
	if err := cmd.PersistentFlags().MarkHidden(optionNameDebugAPIURLTemplate); err != nil {
		panic(err)
	}
	cmd.PersistentFlags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace")
	cmd.PersistentFlags().IntP(optionNameNodeCount, "c", 1, "node count")

	cmd.AddCommand(c.initCheckPeerCount())
	cmd.AddCommand(c.initCheckPingPong())
	cmd.AddCommand(c.initCheckPushSync())

	c.root.AddCommand(cmd)
	return nil
}
