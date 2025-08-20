package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/ethersphere/beekeeper/pkg/nuker"
	"github.com/spf13/cobra"
)

const (
	optionNameRestartArgs            = "restart-args"
	optionNameUseRandomNeighboorhood = "use-random-neighborhood"
	optionNameDeploymentType         = "deployment-type"
	beeLabelSelector                 = "app.kubernetes.io/name=bee"
)

func (c *command) initNukeCmd() (err error) {
	cmd := &cobra.Command{
		Use:     "nuke",
		Short:   "Clears databases and restarts Bee.",
		Example: `beekeeper nuke --cluster-name=default --restart-args="bee,start,--config=.bee.yaml"`,
		Long: `Executes a database nuke operation across Bee nodes in a Kubernetes cluster, forcing each node to resynchronize all data on next startup.
		This command provides StatefulSet update and rollback procedures to maintain cluster stability during the nuke process, ensuring safe and coordinated resets of node state.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
				nodeClient, err := c.createNodeClient(ctx, true)
				if err != nil {
					return fmt.Errorf("creating node client: %w", err)
				}

				nukerClient := nuker.New(&nuker.ClientConfig{
					Log:                   c.log,
					K8sClient:             c.k8sClient,
					NodeProvider:          nodeClient,
					UseRandomNeighborhood: c.globalConfig.GetBool(optionNameUseRandomNeighboorhood),
				})

				if err := nukerClient.Run(ctx, c.globalConfig.GetStringSlice(optionNameRestartArgs)); err != nil {
					return fmt.Errorf("running nuke command: %w", err)
				}

				return nil
			})
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name.")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name).")
	cmd.Flags().String(optionNameLabelSelector, beeLabelSelector, "Kubernetes label selector for filtering resources when namespace is set (use empty string for all).")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "Timeout")
	cmd.Flags().StringSlice(optionNameRestartArgs, []string{"bee", "start", "--config=.bee.yaml"}, "Command to run in the Bee cluster, e.g. 'db,nuke,--config=.bee.yaml'")
	cmd.Flags().Bool(optionNameUseRandomNeighboorhood, false, "Use random neighborhood for Bee nodes (default: false)")
	cmd.Flags().String(optionNameDeploymentType, string(node.DeploymentTypeBeekeeper), "Indicates how the cluster was deployed: 'beekeeper' or 'helm'.")

	c.root.AddCommand(cmd)

	return nil
}

func isValidDeploymentType(dt string) bool {
	return dt == string(node.DeploymentTypeHelm) || dt == string(node.DeploymentTypeBeekeeper)
}
