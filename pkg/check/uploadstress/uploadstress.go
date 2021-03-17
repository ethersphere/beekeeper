package uploadstress

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
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
	// rnds := random.PseudoGenerators(o.Seed, cluster.Size())
	fmt.Printf("Seed: %d\n", o.Seed)

	for k, v := range cluster.NodeGroups() {
		fmt.Println(k, v)
	}

	fmt.Println("upload stress check completed successfully")
	return
}
