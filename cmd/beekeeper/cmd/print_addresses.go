package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initPrintAddresses() *cobra.Command {
	return &cobra.Command{
		Use:   "addresses",
		Short: "Print addresses",
		Long:  `Print address for every node in a cluster`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cluster, err := bee.NewCluster(bee.ClusterOptions{
				APIScheme:               c.config.GetString(optionNameAPIScheme),
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				APIInsecureTLS:          insecureTLSAPI,
				DebugAPIScheme:          c.config.GetString(optionNameDebugAPIScheme),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				DebugAPIInsecureTLS:     insecureTLSDebugAPI,
				Namespace:               c.config.GetString(optionNameNamespace),
				Size:                    c.config.GetInt(optionNameNodeCount),
			})
			if err != nil {
				return err
			}

			ctx := context.Background()

			return printAddresses(ctx, cluster)
		},
		PreRunE: c.printPreRunE,
	}
}

func printAddresses(ctx context.Context, cluster bee.Cluster) (err error) {
	for i, n := range cluster.Nodes {
		a, err := n.Addresses(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("%d. %+v\n", i, a)
	}
	return
}
