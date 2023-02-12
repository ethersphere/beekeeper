package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/spf13/cobra"
)

func (c *command) initPrintCmd() (err error) {
	const (
		optionNameClusterName = "cluster-name"
		optionNameTimeout     = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "print",
		Short: "prints information about a Bee cluster",
		Long: `Prints information about a Bee cluster: addresses, depths, nodes, overlays, peers, topologies
Requires exactly one argument from the following list: addresses, depths, nodes, overlays, peers, topologies`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly one argument from the following list: addresses, depths, nodes, overlays, peers, topologies, config")
			}

			if _, ok := printFuncs[args[0]]; !ok {
				return fmt.Errorf("argument '%s' is not from the following list: addresses, depths, nodes, overlays, peers, topologies, config", args[0])
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if args[0] == "config" {
				if err := c.config.PrintYaml(os.Stdout); err != nil {
					return fmt.Errorf("config can not be printed: %s", err.Error())
				}
				return
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			cluster, err := c.setupCluster(ctx, c.globalConfig.GetString(optionNameClusterName), c.config, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			f, ok := printFuncs[args[0]]
			if !ok {
				return fmt.Errorf("printing %s not implemented", args[0])
			}

			return f(ctx, cluster)
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if args[0] == "config" {
				return
			}
			return c.preRunE(cmd, args)
		},
	}

	cmd.PersistentFlags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().Duration(optionNameTimeout, 15*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

var printFuncs = map[string]func(ctx context.Context, cluster orchestration.Cluster) (err error){
	"addresses": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		addresses, err := cluster.Addresses(ctx)
		if err != nil {
			return err
		}

		for ng, na := range addresses {
			fmt.Printf("Printing %s node group's addresses\n", ng)
			for n, a := range na {
				fmt.Printf("Node %s. ethereum: %s\n", n, a.Ethereum)
				fmt.Printf("Node %s. public key: %s\n", n, a.PublicKey)
				fmt.Printf("Node %s. overlay: %s\n", n, a.Overlay)
				for _, u := range a.Underlay {
					fmt.Printf("Node %s. underlay: %s\n", n, u)
				}
			}
		}

		return
	},
	"depths": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		topologies, err := cluster.Topologies(ctx)
		if err != nil {
			return err
		}

		for ng, nt := range topologies {
			fmt.Printf("Printing %s node group's topologies\n", ng)
			for n, t := range nt {
				fmt.Printf("Node %s. overlay: %s depth: %d\n", n, t.Overlay, t.Depth)
			}
		}

		return
	},
	"nodes": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		nodes := cluster.NodeNames()

		for _, n := range nodes {
			fmt.Printf("%s\n", n)
		}

		return
	},
	"overlays": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		overlays, err := cluster.Overlays(ctx)
		if err != nil {
			return err
		}

		for ng, no := range overlays {
			fmt.Printf("Printing %s node group's overlays\n", ng)
			for n, o := range no {
				fmt.Printf("Node %s. %s\n", n, o.String())
			}
		}

		return
	},
	"peers": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		peers, err := cluster.Peers(ctx)
		if err != nil {
			return err
		}

		for ng, np := range peers {
			fmt.Printf("Printing %s node group's peers\n", ng)
			for n, a := range np {
				for _, p := range a {
					fmt.Printf("Node %s. %s\n", n, p)
				}
			}
		}
		return
	},
	"topologies": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		topologies, err := cluster.Topologies(ctx)
		if err != nil {
			return err
		}

		for ng, nt := range topologies {
			fmt.Printf("Printing %s node group's topologies\n", ng)
			for n, t := range nt {
				fmt.Printf("Node %s. overlay: %s\n", n, t.Overlay)
				fmt.Printf("Node %s. population: %d\n", n, t.Population)
				fmt.Printf("Node %s. connected: %d\n", n, t.Connected)
				fmt.Printf("Node %s. depth: %d\n", n, t.Depth)
				fmt.Printf("Node %s. nnLowWatermark: %d\n", n, t.NnLowWatermark)
				for k, v := range t.Bins {
					fmt.Printf("Node %s. %s %+v\n", n, k, v)
				}
			}
		}

		return
	},
	"config": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		return
	},
}
