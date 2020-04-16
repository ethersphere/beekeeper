package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ethersphere/bee/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/peercount"
)

func (c *command) initCheckCmd() (err error) {

	const (
		optionNameNodeCount       = "node-count"
		optionNameNodeURLTemplate = "node-url-template"
		optionNameVerbosity       = "verbosity"
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check Bees",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) > 0 {
				return cmd.Help()
			}

			var logger logging.Logger
			switch v := strings.ToLower(c.config.GetString(optionNameVerbosity)); v {
			case "0", "silent":
				logger = logging.New(ioutil.Discard, 0)
			case "1", "error":
				logger = logging.New(cmd.OutOrStdout(), logrus.ErrorLevel)
			case "2", "warn":
				logger = logging.New(cmd.OutOrStdout(), logrus.WarnLevel)
			case "3", "info":
				logger = logging.New(cmd.OutOrStdout(), logrus.InfoLevel)
			case "4", "debug":
				logger = logging.New(cmd.OutOrStdout(), logrus.DebugLevel)
			case "5", "trace":
				logger = logging.New(cmd.OutOrStdout(), logrus.TraceLevel)
			default:
				return fmt.Errorf("unknown verbosity level %q", v)
			}

			logger.Info("beekeeper check")
			peercount.Check(peercount.Options{
				NodeCount:       c.config.GetInt(optionNameNodeCount),
				NodeURLTemplate: c.config.GetString(optionNameNodeURLTemplate),
			})

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return c.config.BindPFlags(cmd.Flags())
		},
	}

	cmd.Flags().Int(optionNameNodeCount, 1, "bee node count")
	cmd.Flags().String(optionNameNodeURLTemplate, "", "bee node URL template")
	cmd.Flags().String(optionNameVerbosity, "info", "log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace")

	c.root.AddCommand(cmd)
	return nil
}
