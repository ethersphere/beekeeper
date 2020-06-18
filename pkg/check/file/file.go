package file

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents pushsync check options
type Options struct {
	UploadNodeCount int
	FilesPerNode    int
	FileName        string
	FileSize        int
	Seed            int64
}

var errFile = errors.New("file")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func Check(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	// overlays, err := c.Overlays(ctx)
	// if err != nil {
	// 	return err
	// }

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.FilesPerNode; j++ {
			file, err := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)
			if err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			if err := c.Nodes[i].UploadFile(ctx, &file); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}

			_, err = c.Nodes[c.Size()-1].DownloadFile(ctx, file.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", c.Size()-1, err)
			}

			fmt.Printf("%s %s %d\n", file.Name(), file.Address().String(), file.Size())
		}
	}

	return
}
