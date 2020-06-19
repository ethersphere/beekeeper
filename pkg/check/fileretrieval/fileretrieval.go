package fileretrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	FilesPerNode    int
	FileName        string
	FileSize        int64
	Seed            int64
}

var errFileRetrieval = errors.New("file retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			if err := c.Nodes[i].UploadFile(ctx, &file); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			time.Sleep(1 * time.Second)
			size, hash, err := c.Nodes[c.Size()-1].DownloadFile(ctx, file.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", c.Size()-1, err)
			}

			if !bytes.Equal(file.Hash(), hash) {
				fmt.Printf("Node %d. File %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s\n", i, j, file.Size(), size, overlays[i].String(), file.Address().String())
				return errFileRetrieval
			}

			fmt.Printf("Node %d. File %d retrieved successfully. Node: %s File: %s\n", i, j, overlays[i].String(), file.Address().String())
		}
	}

	return
}
