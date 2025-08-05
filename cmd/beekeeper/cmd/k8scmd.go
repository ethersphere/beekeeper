package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/k8scmd"
	"github.com/spf13/cobra"
)

const optionNameArgs = "args"

func (c *command) initK8sCmd() (err error) {
	cmd := &cobra.Command{
		Use:     "k8scmd",
		Short:   "k8scmd udpates bee command in the stateful set",
		Example: `beekeeper k8scmd --cluster-name=default --args="bee,start,--config=.bee.yaml"`,
		Long:    `update Bee command in the stateful set.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				clusterName := c.globalConfig.GetString(optionNameClusterName)
				if clusterName == "" {
					return errMissingClusterName
				}

				if !c.globalConfig.IsSet(optionNameArgs) {
					return fmt.Errorf("no command specified to update Bee cluster %s", clusterName)
				}

				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("setting up cluster %s: %w", clusterName, err)
				}

				beeClients, err := cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to retrieve node clients: %w", err)
				}

				commander := k8scmd.New(&k8scmd.ClientConfig{
					Log:        c.log,
					K8sClient:  c.k8sClient,
					BeeClients: beeClients,
				})

				if _, err := commander.Run(ctx, c.globalConfig.GetStringSlice(optionNameArgs)); err != nil {
					return fmt.Errorf("updating Bee cluster %s: %w", clusterName, err)
				}

				return nil
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "cluster name")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")
	cmd.Flags().String(optionNameArgs, "", "command to run in the Bee cluster, e.g. 'db,nuke,--config=.bee.yaml'")

	c.root.AddCommand(cmd)

	return nil
}
