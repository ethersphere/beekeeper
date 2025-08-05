package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8scmd"
	"github.com/spf13/cobra"
)

const (
	optionNameArgs           = "args"
	optionNameUpdateStrategy = "update-strategy"
	beeLabelSelector         = "app.kubernetes.io/name=bee"
)

func (c *command) initK8sCmd() (err error) {
	cmd := &cobra.Command{
		Use:     "k8scmd",
		Short:   "k8scmd udpates bee command in the stateful set",
		Example: `beekeeper k8scmd --cluster-name=default --args="bee,start,--config=.bee.yaml"`,
		Long:    `update Bee command in the stateful set.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				if !c.globalConfig.IsSet(optionNameArgs) {
					return errors.New("args must be provided")
				}

				// Validate update strategy
				updateStrategy := c.globalConfig.GetString(optionNameUpdateStrategy)
				if updateStrategy != "RollingUpdate" && updateStrategy != "OnDelete" {
					return fmt.Errorf("invalid update strategy '%s': must be 'RollingUpdate' or 'OnDelete'", updateStrategy)
				}

				namespace := c.globalConfig.GetString(optionNameNamespace)
				clusterName := c.globalConfig.GetString(optionNameClusterName)

				if clusterName == "" && namespace == "" {
					return errors.New("either cluster name or namespace must be provided")
				}

				var beeClients map[string]*bee.Client

				if clusterName != "" {
					cluster, err := c.setupCluster(ctx, clusterName, false)
					if err != nil {
						return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
					}

					beeClients, err = cluster.NodesClients(ctx)
					if err != nil {
						return fmt.Errorf("failed to retrieve node clients: %w", err)
					}

					namespace = cluster.Namespace()
				}

				commander := k8scmd.New(&k8scmd.ClientConfig{
					Log:            c.log,
					K8sClient:      c.k8sClient,
					BeeClients:     beeClients,
					UpdateStrategy: updateStrategy,
				})

				if err := commander.Run(ctx, namespace, c.globalConfig.GetString(optionNameLabelSelector), c.globalConfig.GetStringSlice(optionNameArgs)); err != nil {
					return fmt.Errorf("updating Bee cluster %s: %w", clusterName, err)
				}

				return nil
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name.")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name).")
	cmd.Flags().String(optionNameLabelSelector, beeLabelSelector, "Kubernetes label selector for filtering resources when namespace is set (use empty string for all).")
	cmd.Flags().String(optionNameUpdateStrategy, "OnDelete", "StatefulSet update strategy: 'RollingUpdate' or 'OnDelete'")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")
	cmd.Flags().StringSlice(optionNameArgs, []string{"bee", "start", "--config=.bee.yaml"}, "command to run in the Bee cluster, e.g. 'db,nuke,--config=.bee.yaml'")

	c.root.AddCommand(cmd)

	return nil
}
