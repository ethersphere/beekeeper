package uploadstress

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// compile check whether UploadStress implements interface
var _ check.Check = (*UploadStress)(nil)

// UploadStress check
type UploadStress struct{}

// NewUploadStress returns new ping check
func NewUploadStress() *UploadStress {
	return &UploadStress{}
}

// Run executes upload stress check
func (u *UploadStress) Run(ctx context.Context, cluster *bee.Cluster, o check.Options) (err error) {
	rnds := random.PseudoGenerators(o.Seed, cluster.Size())
	fmt.Printf("Seed: %d\n", o.Seed)

	for _, ng := range cluster.NodeGroups() {
		for _, n := range ng.Nodes() {
			o, err := n.Client().Overlay(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", n.Name(), err)
			}
			fmt.Println("overlay", n.Name(), o)

			file := bee.NewRandomFile(rnds[0], "filename", 1)
			tagResponse, err := n.Client().CreateTag(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", n.Name(), err)
			}
			tagUID := tagResponse.Uid

			if err := n.Client().UploadFileWithTag(ctx, &file, false, tagUID); err != nil {
				return fmt.Errorf("node %s: %w", n.Name(), err)
			}

			fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), o)
		}
	}

	fmt.Println("upload stress check completed successfully")
	return
}
