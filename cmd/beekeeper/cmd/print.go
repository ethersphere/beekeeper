package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging"
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
		Long: `Prints information about a Bee cluster: addresses, depths, overlays, peers, topologies
Requires exactly one argument from the following list: addresses, depths, overlays, peers, topologies`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires exactly one argument from the following list: addresses, depths, overlays, peers, topologies")
			}

			for k := range printFuncs {
				if k == args[0] {
					return nil
				}
			}

			return fmt.Errorf("requires exactly one argument from the following list: addresses, depths, overlays, peers, topologies")
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
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

			return f(ctx, cluster, c.logger)
		},
		PreRunE: c.preRunE,
	}

	cmd.PersistentFlags().String(optionNameClusterName, "default", "cluster name")
	cmd.Flags().Duration(optionNameTimeout, 15*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

var printFuncs = map[string]func(ctx context.Context, cluster orchestration.Cluster, logger logging.Logger) (err error){
	"addresses": func(ctx context.Context, cluster orchestration.Cluster, logger logging.Logger) (err error) {
		addresses, err := cluster.Addresses(ctx)
		if err != nil {
			return err
		}

		for ng, na := range addresses {
			logger.Infof("Printing %s node group's addresses", ng)
			for n, a := range na {
				logger.Infof("Node %s. ethereum: %s", n, a.Ethereum)
				logger.Infof("Node %s. public key: %s", n, a.PublicKey)
				logger.Infof("Node %s. overlay: %s", n, a.Overlay)
				for _, u := range a.Underlay {
					logger.Infof("Node %s. underlay: %s", n, u)
				}
			}
		}

		return
	},
	"depths": func(ctx context.Context, cluster orchestration.Cluster, logger logging.Logger) (err error) {
		topologies, err := cluster.Topologies(ctx)
		if err != nil {
			return err
		}

		for ng, nt := range topologies {
			logger.Infof("Printing %s node group's topologies", ng)
			for n, t := range nt {
				logger.Infof("Node %s. overlay: %s depth: %d", n, t.Overlay, t.Depth)
			}
		}

		return
	},
	"overlays": func(ctx context.Context, cluster orchestration.Cluster, logger logging.Logger) (err error) {
		overlays, err := cluster.Overlays(ctx)
		if err != nil {
			return err
		}

		for ng, no := range overlays {
			logger.Infof("Printing %s node group's overlays", ng)
			for n, o := range no {
				logger.Infof("Node %s. %s", n, o.String())
			}
		}

		return
	},
	"peers": func(ctx context.Context, cluster orchestration.Cluster, logger logging.Logger) (err error) {
		peers, err := cluster.Peers(ctx)
		if err != nil {
			return err
		}

		for ng, np := range peers {
			logger.Infof("Printing %s node group's peers", ng)
			for n, a := range np {
				for _, p := range a {
					logger.Infof("Node %s. %s", n, p)
				}
			}
		}
		return
	},
	"topologies": func(ctx context.Context, cluster orchestration.Cluster, logger logging.Logger) (err error) {
		topologies, err := cluster.Topologies(ctx)
		if err != nil {
			return err
		}

		for ng, nt := range topologies {
			logger.Infof("Printing %s node group's topologies", ng)
			for n, t := range nt {
				logger.Infof("Node %s. overlay: %s", n, t.Overlay)
				logger.Infof("Node %s. population: %d", n, t.Population)
				logger.Infof("Node %s. connected: %d", n, t.Connected)
				logger.Infof("Node %s. depth: %d", n, t.Depth)
				logger.Infof("Node %s. nnLowWatermark: %d", n, t.NnLowWatermark)
				for k, v := range t.Bins {
					logger.Infof("Node %s. %s %+v", n, k, v)
				}
			}
		}

		return
	},
}
