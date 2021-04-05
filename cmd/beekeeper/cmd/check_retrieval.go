package cmd

import (
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/retrieval"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/spf13/cobra"
)

func (c *command) initCheckRetrieval() *cobra.Command {
	const (
		optionNameUploadNodeCount = "upload-node-count"
		optionNameChunksPerNode   = "chunks-per-node"
		optionNameSeed            = "seed"
	)

	cmd := &cobra.Command{
		Use:   "retrieval",
		Short: "Checks retrieval ability of the cluster",
		Long: `Checks retrieval ability of the cluster.
It uploads given number of chunks to given number of nodes, 
and attempts retrieval of those chunks from the last node in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameUploadNodeCount) > c.config.GetInt(optionNameNodeCount) {
				return errors.New("bad parameters: upload-node-count must be less or equal to node-count")
			}

			cfg := config.Read("config.yaml")

			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), c.config.GetString(optionNameNamespace))

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return retrieval.Check(cluster, retrieval.Options{
				NodeGroup:       "bee",
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
				ChunksPerNode:   c.config.GetInt(optionNameChunksPerNode),
				Seed:            seed,
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload chunks to")
	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of chunks to upload per node")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")

	return cmd
}
