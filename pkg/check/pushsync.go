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
	DebugAPIHostnamePattern string
	DebugAPIDomain          string
	DisableNamespace        bool
	Namespace               string
	NodeCount               int
	RandomSeed              bool
	Seed                    int64
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

	for i := 0; i < opts.NodeCount; i++ {
		chunkSize := rand.Intn(maxChunkSize)
		fmt.Printf("chunkSize: %d\n", chunkSize)

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

		fmt.Printf("Node %d. Hash: %s\n", i, r.Hash)

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

		fmt.Printf("Node %d. Hash: %s\n", i, resp.Message)
	}

	return
}
