package check

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

// PushSyncOptions ...
type PushSyncOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	Namespace               string
	NodeCount               int

	ChunksPerNode   int
	TargetNode      int
	UploadNodeCount int

	RandomSeed bool
	Seed       int64
}

var errPushSync = errors.New("pushsync")

// PushSync ...
func PushSync(opts PushSyncOptions) (err error) {
	var seed int64
	if opts.RandomSeed {
		var src cryptoSource
		rnd := rand.New(src)
		seed = rnd.Int63()
	} else {
		seed = opts.Seed
	}
	fmt.Printf("seed: %d\n", seed)

	chunks, err := generateChunks(opts.UploadNodeCount, opts.ChunksPerNode, seed)
	if err != nil {
		return err
	}

	ctx := context.Background()
	nodes, err := bee.NewNNodes(opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, opts.DisableNamespace, opts.NodeCount)
	if err != nil {
		return err
	}

	var overlays []swarm.Address
	for _, n := range nodes {
		a, err := n.DebugAPI.Node.Addresses(ctx)
		if err != nil {
			return err
		}

		overlays = append(overlays, a.Overlay)
	}

	testFailed := false
	uploadNodes := nodes[:opts.UploadNodeCount]
	for i, n := range uploadNodes {
		fmt.Printf("Node %d:\n", i)
		for j := 0; j < opts.ChunksPerNode; j++ {
			// make data
			chunk := chunks[i][j]
			data := bytes.NewReader(chunk.Data)
			fmt.Printf("Chunk %d size: %d\n", j, len(chunk.Data))

			// upload data
			r, err := n.API.Bzz.Upload(ctx, data)
			if err != nil {
				return err
			}
			chunk.Address = r.Hash
			fmt.Printf("Chunk %d hash: %s\n", j, chunk.Address)

			// find chunk's closest node
			closestNode, err := chunk.ClosestNode(overlays)
			if err != nil {
				return err
			}
			closestIndex := findIndex(overlays, closestNode)
			fmt.Printf("Chunk %d closest node: %s\n", j, closestNode)

			time.Sleep(1 * time.Second)
			// check
			resp, err := nodes[closestIndex].DebugAPI.Node.HasChunk(ctx, chunk.Address)
			if resp.Message == "OK" {
				fmt.Printf("Chunk %d found on closest node\n", j)
			} else if err == debugapi.ErrNotFound {
				fmt.Printf("Chunk %d not found on closest node\n", j)
				testFailed = true
			} else if err != nil {
				return err
			}
		}
	}

	if testFailed {
		return errPushSync
	}

	return
}

func findIndex(overlays []swarm.Address, addr swarm.Address) int {
	for i, a := range overlays {
		if addr.Equal(a) {
			return i
		}
	}
	return -1
}

// generateChunks generates chunks for nodes
func generateChunks(nodeCount, chunksPerNode int, seed int64) (chunks map[int]map[int]bee.Chunk, err error) {
	randomChunks, err := bee.NewNRandomChunks(seed, nodeCount*chunksPerNode)
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
