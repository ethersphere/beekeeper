package check

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee/api"
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

		data := make([]byte, chunkSize)
		if _, err := rand.Read(data); err != nil {
			return err
		}

		APIURL, err := createURL(scheme, opts.APIHostnamePattern, opts.Namespace, opts.APIDomain, i, opts.DisableNamespace)
		if err != nil {
			return err
		}

		c := api.NewClient(APIURL, nil)
		ctx := context.Background()

		fmt.Println(data)
		r, err := c.Bzz.Upload(ctx, bytes.NewReader(data))
		if err != nil {
			return err
		}

		fmt.Printf("Node %d. Hash: %s\n", i, r.Hash)
	}

	return
}
