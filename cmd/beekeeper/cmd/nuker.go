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
	beeLabelSelector                 = "app.kubernetes.io/name=bee"
)

func (c *command) initNukeCmd() (err error) {
	cmd := &cobra.Command{
		Use:     "nuke",
		Short:   "nuke",
		Example: `beekeeper nuke --cluster-name=default --restart-args="bee,start,--config=.bee.yaml"`,
		Long:    `update Bee command in the stateful set.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.withTimeoutHandler(cmd, func(ctx context.Context) error {
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

				nodeClient := node.New(&node.ClientConfig{
					Log:           c.log,
					K8sClient:     c.k8sClient,
					Namespace:     namespace,
					HTTPClient:    c.httpClient,
					LabelSelector: c.globalConfig.GetString(optionNameLabelSelector),
					InCluster:     c.globalConfig.GetBool(optionNameInCluster),
					BeeClients:    beeClients,
					UseNamespace:  c.globalConfig.IsSet(optionNameNamespace),
				})

				nodes, err := nodeClient.GetNodes(ctx)
				if err != nil {
					return fmt.Errorf("getting nodes: %w", err)
				}

				var neighborhoodArgProvider nuker.NeighborhoodArgProvider
				if c.globalConfig.GetBool(optionNameUseRandomNeighboorhood) {
					neighborhoodArgProvider = nuker.NewRandomNeighborhoodProvider(c.log, nodes)
				} else {
					neighborhoodArgProvider = &nuker.NeighborhoodArgProviderNotSet{}
				}

				nuker := nuker.New(&nuker.ClientConfig{
					Log:                     c.log,
					K8sClient:               c.k8sClient,
					BeeClients:              beeClients,
					NeighborhoodArgProvider: neighborhoodArgProvider,
				})

				if err := nuker.Run(ctx, namespace, c.globalConfig.GetString(optionNameLabelSelector), c.globalConfig.GetStringSlice(optionNameRestartArgs)); err != nil {
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
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")
	cmd.Flags().StringSlice(optionNameRestartArgs, []string{"bee", "start", "--config=.bee.yaml"}, "command to run in the Bee cluster, e.g. 'db,nuke,--config=.bee.yaml'")
	cmd.Flags().Bool(optionNameUseRandomNeighboorhood, false, "use random neighborhood for Bee nodes (default: false)")

	c.root.AddCommand(cmd)

	return nil
}
