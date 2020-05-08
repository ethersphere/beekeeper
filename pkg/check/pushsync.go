package check

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

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
	rand.Seed(seed)
	fmt.Printf("seed: %d\n", seed)

	for i := 0; i < opts.UploadNodeCount; i++ {
		fmt.Printf("Node %d:\n", i)
		for j := 0; j < opts.ChunksPerNode; j++ {
			chunkSize := rand.Intn(maxChunkSize)
			fmt.Printf("Chunk %d size: %d\n", j, chunkSize)

			chunk := make([]byte, chunkSize)
			if _, err := rand.Read(chunk); err != nil {
				return err
			}

			APIURL, err := createURL(scheme, opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, i, opts.DisableNamespace)
			if err != nil {
				return err
			}

			c := api.NewClient(APIURL, nil)
			ctx := context.Background()
			data := bytes.NewReader(chunk)

			r, err := c.Bzz.Upload(ctx, data)
			if err != nil {
				return err
			}

			fmt.Printf("Chunk %d hash: %s\n", j, r.Hash)

			debugAPIURL, err := createURL(scheme, opts.DebugAPIHostnamePattern, opts.Namespace, opts.DebugAPIDomain, i, opts.DisableNamespace)
			if err != nil {
				return err
			}

			dc := debugapi.NewClient(debugAPIURL, nil)
			ctx = context.Background()

			resp, err := dc.Node.HasChunk(ctx, r.Hash)
			if err != nil {
				return err
			}

			if resp.Message == "OK" {
				fmt.Printf("Chunk %d found\n", j)
			} else {
				fmt.Printf("Chunk %d not found\n", j)
			}

		}
	}

	return
}
