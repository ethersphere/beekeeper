package cmd

import (
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPingPong() *cobra.Command {
	return &cobra.Command{
		Use:   "pingpong",
		Short: "Checks pingpong",
		Long:  `Checks pingpong`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			nodes, err := bee.NewNNodes(
				c.config.GetString(optionNameAPIHostnamePattern),
				c.config.GetString(optionNameNamespace),
				c.config.GetString(optionNameAPIDomain),
				c.config.GetString(optionNameDebugAPIHostnamePattern),
				c.config.GetString(optionNameNamespace),
				c.config.GetString(optionNameDebugAPIDomain),
				disableNamespace,
				c.config.GetInt(optionNameNodeCount),
			)
			if err != nil {
				return err
			}

			return check.PingPong(nodes)
		},
		PreRunE: c.checkPreRunE,
	}
}
