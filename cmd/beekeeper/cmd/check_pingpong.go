package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPingPong() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pingpong",
		Short: "Checks pingpong",
		Long:  `Checks pingpong`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return check.PingPong(check.PingPongOptions{
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				Namespace:               c.config.GetString(optionNameNamespace),
				NodeCount:               c.config.GetInt(optionNameNodeCount),
			})
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	return cmd
}
