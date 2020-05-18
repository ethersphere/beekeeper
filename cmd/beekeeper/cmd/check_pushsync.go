package cmd

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/pushsync"

	"github.com/spf13/cobra"
)

func (c *command) initCheckPushSync() *cobra.Command {
	const (
		optionNameChunksPerNode   = "chunks-per-node"
		optionNameRandomSeed      = "random-seed"
		optionNameSeed            = "seed"
		optionNameUploadNodeCount = "upload-node-count"
	)

	cmd := &cobra.Command{
		Use:   "pushsync",
		Short: "Checks push sync",
		Long:  `Checks push sync`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameUploadNodeCount) > c.config.GetInt(optionNameNodeCount) {
				return errors.New("upload-node-count must be less or equal to node-count")
			}
			var seed int64
			if c.config.GetBool(optionNameRandomSeed) {
				var src cryptoSource
				rnd := rand.New(src)
				seed = rnd.Int63()
			} else {
				seed = c.config.GetInt64(optionNameSeed)
			}
			fmt.Printf("seed: %d\n", seed)

			chunks, err := generateChunks(
				c.config.GetInt(optionNameUploadNodeCount),
				c.config.GetInt(optionNameChunksPerNode),
				seed,
			)
			if err != nil {
				return err
			}

			cluster, err := bee.NewCluster(bee.ClusterOptions{
				APIHostnamePattern:      c.config.GetString(optionNameAPIHostnamePattern),
				APIDomain:               c.config.GetString(optionNameAPIDomain),
				DebugAPIHostnamePattern: c.config.GetString(optionNameDebugAPIHostnamePattern),
				DebugAPIDomain:          c.config.GetString(optionNameDebugAPIDomain),
				Namespace:               c.config.GetString(optionNameNamespace),
				Size:                    c.config.GetInt(optionNameNodeCount),
			})
			if err != nil {
				return err
			}

			return pushsync.Check(cluster, chunks)
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameChunksPerNode, "p", 1, "number of chunks to upload per node")
	cmd.Flags().BoolP(optionNameRandomSeed, "r", true, "random seed")
	cmd.Flags().Int64P(optionNameSeed, "s", 1, "seed")
	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload chunks to")

	return cmd
}

// cryptoSource is used to create truly random source
type cryptoSource struct{}

func (s cryptoSource) Seed(seed int64) {}

func (s cryptoSource) Int63() int64 {
	return int64(s.Uint64() & ^uint64(1<<63))
}

func (s cryptoSource) Uint64() (v uint64) {
	err := binary.Read(crand.Reader, binary.BigEndian, &v)
	if err != nil {
		log.Fatal(err)
	}
	return v
}

// generateChunks generates chunks for nodes
func generateChunks(nodeCount, chunksPerNode int, seed int64) (chunks map[int]map[int]bee.Chunk, err error) {
	randomChunks, err := bee.NewRandomChunks(seed, nodeCount*chunksPerNode)
	if err != nil {
		return map[int]map[int]bee.Chunk{}, err
	}

	chunks = make(map[int]map[int]bee.Chunk)
	for i := 0; i < nodeCount; i++ {
		tmp := randomChunks[0:chunksPerNode]

		nodeChunks := make(map[int]bee.Chunk)
		for j := 0; j < chunksPerNode; j++ {
			nodeChunks[j] = tmp[j]
		}

		chunks[i] = nodeChunks
		randomChunks = randomChunks[chunksPerNode:]
	}

	return
}
