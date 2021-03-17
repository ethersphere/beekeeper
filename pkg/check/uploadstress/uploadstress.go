package uploadstress

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/random"
	"golang.org/x/sync/errgroup"
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
	buffer := 5
	ngGroup := new(errgroup.Group)
	ngSemaphore := make(chan struct{}, buffer)
	for _, ng := range cluster.NodeGroups() {
		ng := ng
		rnds := random.PseudoGenerators(o.Seed, ng.Size())

		nGroup := new(errgroup.Group)
		nSemaphore := make(chan struct{}, buffer)

		ngSemaphore <- struct{}{}
		ngGroup.Go(func() error {
			defer func() {
				<-ngSemaphore
			}()

			nodes := ng.NodesSorted()
			for i := 0; i < ng.Size(); i++ {
				i := i
				n := ng.Node(nodes[i])

				nSemaphore <- struct{}{}
				nGroup.Go(func() error {
					defer func() {
						<-nSemaphore
					}()

					for {
						o, err := n.Client().Overlay(ctx)
						if err != nil {
							return fmt.Errorf("node %s: %w", n.Name(), err)
						}

						file := bee.NewRandomFile(rnds[i], "filename", 1)
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
				})
			}

			return nGroup.Wait()
		})
	}

	if err := ngGroup.Wait(); err != nil {
		return err
	}

	fmt.Println("upload stress check completed successfully")
	return
}
