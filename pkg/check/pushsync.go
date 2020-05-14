package check

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/bee/debugapi"
)

// PushSyncOptions ...
type PushSyncOptions struct {
	APIHostnamePattern      string
	APIDomain               string
	ChunksPerNode           int
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	Namespace               string
	NodeCount               int
	RandomSeed              bool
	Seed                    int64
	TargetNode              int
	UploadNodeCount         int
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

	overlayAddresses, err := getOverlayAddresses(opts.NodeCount, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, opts.DisableNamespace)
	if err != nil {
		return err
	}

	allNodes, err := NewNNodes(opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, opts.DisableNamespace, opts.NodeCount)
	if err != nil {
		return err
	}
	fmt.Println(allNodes)

	testFailed := false
	for i := 0; i < opts.UploadNodeCount; i++ {
		fmt.Printf("Node %d:\n", i)
		for j := 0; j < opts.ChunksPerNode; j++ {
			// select chunk
			chunk := chunks[i][j]
			fmt.Printf("Chunk %d size: %d\n", j, len(chunk.Data))

			// upload chunk
			APIURL, err := createURL(scheme, opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, i, opts.DisableNamespace)
			if err != nil {
				return err
			}

			c := api.NewClient(APIURL, nil)
			ctx := context.Background()
			data := bytes.NewReader(chunk.Data)

			r, err := c.Bzz.Upload(ctx, data)
			if err != nil {
				return err
			}
			chunk.Address = r.Hash
			fmt.Printf("Chunk %d hash: %s\n", j, chunk.Address)

			// finc chunk's closest node
			closestNode, err := chunk.ClosestNode(overlayAddresses)
			if err != nil {
				return err
			}
			fmt.Printf("Chunk %d closest node: %s\n", j, closestNode)

			index := overlayIndex(closestNode, overlayAddresses)
			debugAPIURL, err := createURL(scheme, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, index, opts.DisableNamespace)
			if err != nil {
				return err
			}

			dc := debugapi.NewClient(debugAPIURL, nil)
			ctx = context.Background()

			time.Sleep(1 * time.Second)
			resp, err := dc.Node.HasChunk(ctx, chunk.Address)
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

func overlayIndex(addr swarm.Address, overlayAddresses []swarm.Address) int {
	for i, a := range overlayAddresses {
		if addr.Equal(a) {
			return i
		}
	}
	return -1
}

// generateChunks generates chunks for nodes
func generateChunks(nodeCount, chunksPerNode int, seed int64) (chunks map[int]map[int]Chunk, err error) {
	randomChunks, err := NewNRandomChunks(seed, nodeCount*chunksPerNode)
	if err != nil {
		return map[int]map[int]Chunk{}, err
	}

	chunks = make(map[int]map[int]Chunk)
	for i := 0; i < nodeCount; i++ {
		tmp := randomChunks[0:chunksPerNode]

		nodeChunks := make(map[int]Chunk)
		for j := 0; j < chunksPerNode; j++ {
			nodeChunks[j] = tmp[j]
		}

		chunks[i] = nodeChunks
		randomChunks = randomChunks[chunksPerNode:]
	}

	return
}

// getOverlayAddresses returns overlay addresses of all nodes
func getOverlayAddresses(nodeCount int, debugAPIHostnamePattern, namespace, debugAPIDomain string, disableNamespace bool) (addresses []swarm.Address, err error) {
	for i := 0; i < nodeCount; i++ {
		debugAPIURL, err := createURL(scheme, debugAPIHostnamePattern, namespace, debugAPIDomain, i, disableNamespace)
		if err != nil {
			return []swarm.Address{}, err
		}

		dc := debugapi.NewClient(debugAPIURL, nil)
		ctx := context.Background()

		a, err := dc.Node.Addresses(ctx)
		if err != nil {
			return []swarm.Address{}, err
		}

		addresses = append(addresses, a.Overlay)
	}

	return
}
