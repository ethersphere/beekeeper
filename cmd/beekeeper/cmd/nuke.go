package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
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
				namespace := c.globalConfig.GetString(optionNameNamespace)
				clusterName := c.globalConfig.GetString(optionNameClusterName)

				if clusterName == "" && namespace == "" {
					return errors.New("either cluster name or namespace must be provided")
				}

				if !isValidDeploymentType(c.globalConfig.GetString(optionNameDeploymentType)) {
					return errors.New("unsupported deployment type: must be 'beekeeper' or 'helm'")
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

				nodeClient := node.New(&node.ClientConfig{
					Log:            c.log,
					HTTPClient:     c.httpClient,
					K8sClient:      c.k8sClient,
					BeeClients:     beeClients,
					Namespace:      namespace,
					LabelSelector:  c.globalConfig.GetString(optionNameLabelSelector),
					DeploymentType: node.DeploymentType(c.globalConfig.GetString(optionNameDeploymentType)),
					InCluster:      c.globalConfig.GetBool(optionNameInCluster),
					UseNamespace:   c.globalConfig.IsSet(optionNameNamespace),
				})

				nodes, err := nodeClient.GetNodes(ctx)
				if err != nil {
					return fmt.Errorf("getting nodes: %w", err)
				}

				nukerClient := nuker.New(&nuker.ClientConfig{
					Log:                   c.log,
					K8sClient:             c.k8sClient,
					Nodes:                 nodes,
					UseRandomNeighborhood: c.globalConfig.GetBool(optionNameUseRandomNeighboorhood),
				})

				if err := nukerClient.Run(ctx, namespace, c.globalConfig.GetStringSlice(optionNameRestartArgs)); err != nil {
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
