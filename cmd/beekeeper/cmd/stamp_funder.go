package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	stampfunder "github.com/ethersphere/beekeeper/pkg/funder/stamp"
	"github.com/spf13/cobra"
)

func (c *command) initStampFunderCmd() (err error) {
	const (
		optionNameNamespace     = "namespace"
		optionClusterName       = "cluster-name"
		optionTTLTreshold       = "ttl-treshold"
		optionTopUpTo           = "topup-to"
		optionUsageThreshold    = "usage-threshold"
		optionDiutionDepth      = "dilution-depth"
		optionPeriodicCheck     = "periodic-check"
		optionNameTimeout       = "timeout"
		optionNameLabelSelector = "label-selector"
	)

	cmd := &cobra.Command{
		Use:   "stamp-funder",
		Short: "funds stamp for nodes",
		Long:  `Funds stamp for nodes.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.NodeFunder{}

			namespace := c.globalConfig.GetString(optionNameNamespace)
			clusterName := c.globalConfig.GetString(optionClusterName)

			if namespace != "" {
				cfg.Namespace = namespace
			} else if clusterName != "" {
				cluster, ok := c.config.Clusters[clusterName]
				if !ok {
					return fmt.Errorf("cluster %s not found", clusterName)
				}
				if cluster.Namespace == nil || *cluster.Namespace == "" {
					return fmt.Errorf("cluster %s namespace not provided", clusterName)
				}
				cfg.Namespace = *cluster.Namespace
			} else {
				return errors.New("one of namespace, or valid cluster-name must be provided")
			}

			// add timeout to stamp-funder
			// if timeout is not set, operator will run infinitely
			var ctxNew context.Context
			var cancel context.CancelFunc
			timeout := c.globalConfig.GetDuration(optionNameTimeout)
			if timeout > 0 {
				ctxNew, cancel = context.WithTimeout(cmd.Context(), timeout)
			} else {
				ctxNew = context.Background()
			}
			if cancel != nil {
				defer cancel()
			}

			return stampfunder.NewClient(&stampfunder.ClientConfig{
				Log:           c.log,
				Namespace:     namespace,
				K8sClient:     c.k8sClient,
				LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
			}).Run(ctxNew)
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace. Overrides cluster name if set.")
	cmd.Flags().String(optionClusterName, "", "Name of the Beekeeper cluster to target. Ignored if a namespace is specified, in which case the namespace from the cluster configuration is used.")
	cmd.Flags().Duration(optionTTLTreshold, 5*24*time.Hour, "Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value.")
	cmd.Flags().Duration(optionTopUpTo, 30*24*time.Hour, "Duration to top up the TTL of a stamp to.")
	cmd.Flags().Float64(optionUsageThreshold, 90, "Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value.")
	cmd.Flags().Uint16(optionDiutionDepth, 1, "Number of levels by which to increase the depth of a stamp during dilution.")
	cmd.Flags().String(optionNameLabelSelector, nodeFunderLabelSelector, "Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources.")
	cmd.Flags().Duration(optionNameTimeout, 0*time.Minute, "Maximum duration to wait for the operation to complete. Default is no timeout.")

	c.root.AddCommand(cmd)

	return nil
}
