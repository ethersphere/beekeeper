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
	const optionNameTimeout = "timeout"

	cmd := &cobra.Command{
		Use:   "print",
		Short: "Prints information about a Bee cluster",
		Long: `Prints detailed information about a Bee cluster and its components.

The print command provides insights into various aspects of your cluster:
• addresses: Display Ethereum addresses, public keys, overlays, and underlays for all nodes
• depths: Show the Kademlia depth for each node in the overlay network
• nodes: List all node names in the cluster
• overlays: Display the overlay addresses for each node
• peers: Show peer connections and network topology
• topologies: Display the complete network topology structure
• config: Print the current cluster configuration in YAML format

This command is useful for debugging, monitoring, and understanding your cluster's state.
Requires exactly one argument from the list above.`,
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
			// no need to setup cluster in case of print config
			if args[0] == "config" {
				if err := c.config.PrintYaml(os.Stdout); err != nil {
					return fmt.Errorf("config can not be printed: %s", err.Error())
				}
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			cluster, err := c.setupCluster(ctx, c.globalConfig.GetString(optionNameClusterName), false)
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
			// skip setup in case of print config
			if args[0] == "config" {
				return err
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

		return err
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

		return err
	},
	"nodes": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		nodes := cluster.NodeNames()

		for _, n := range nodes {
			fmt.Printf("%s\n", n)
		}

		return err
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

		return err
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
		return err
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

		return err
	},
	// print config prints configuration used to setup cluster
	// add it to print funcs and do nothing (required to check if argument exists)
	"config": func(ctx context.Context, cluster orchestration.Cluster) (err error) {
		return err
	},
}
