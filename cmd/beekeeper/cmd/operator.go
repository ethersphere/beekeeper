package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

func (c *command) initOperatorCmd() (err error) {
	const (
		optionNameNamespace = "namespace"
		optionNameTimeout   = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "scans for scheduled pods and funds them",
		Long:  `Operator scans for scheduled pods and funds them using node-funder. beekeeper operator`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			namespace := c.globalConfig.GetString(optionNameNamespace)

			// add timeout to operator
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			c.log.Infof("operator started")
			defer c.log.Infof("operator done")

			c.log.Infof("starting events watch")
			err = c.k8sClient.Pods.EventsWatch(ctx, namespace)
			if err != nil {
				c.log.Errorf("events watch: %v", err)
			}
			return nil
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace to scan for scheduled pods.")
	cmd.Flags().Duration(optionNameTimeout, 5*time.Minute, "Timeout.")

	c.root.AddCommand(cmd)

	return nil
}
