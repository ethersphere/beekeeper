package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ethersphere/beekeeper/pkg/check"
)

const (
	optionNameNodeCount       = "node-count"
	optionNameNodeURLTemplate = "node-url-template"
	optionNameVerbosity       = "verbosity"
)

func (c *command) initCheckCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check Bees",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) > 0 {
				return cmd.Help()
			}

			opts := check.Options{
				NodeCount:       c.config.GetInt(optionNameNodeCount),
				NodeURLTemplate: c.config.GetString(optionNameNodeURLTemplate),
			}

			if err = check.Nodes(opts); err != nil {
				return err
			}
			if err = check.PeerCount(opts); err != nil {
				return err
			}

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.PersistentFlags().Int(optionNameNodeCount, 1, "bee node count")
	cmd.PersistentFlags().String(optionNameNodeURLTemplate, "", "bee node URL template")
	cmd.PersistentFlags().String(optionNameVerbosity, "info", "log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace")
	cmd.AddCommand(c.initCheckNodes())
	cmd.AddCommand(c.initCheckPeerCount())

	c.root.AddCommand(cmd)
	return nil
}
