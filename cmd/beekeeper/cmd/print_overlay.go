package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/spf13/cobra"
)

func (c *command) initPrintOverlay() *cobra.Command {
	return &cobra.Command{
		Use:   "overlay",
		Short: "Print overlay addresses",
		Long:  `Print overlay address for every node in a cluster`,
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

			t0 := time.Now()
			ctx := context.Background()
			for o := range cluster.OverlaysStream(ctx) {
				if o.Error != nil {
					fmt.Printf("%d %s\n", o.Index, o.Error)
					continue
				}
				fmt.Printf("%d %s\n", o.Index, o.Address)
			}
			t1 := time.Since(t0)
			fmt.Printf("Overlay took %s\n", t1)

			return
		},
		PreRunE: c.printPreRunE,
	}
}
