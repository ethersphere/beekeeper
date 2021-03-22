package uploadstress

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

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
	buffer := 10

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("node clients: %w", err)
	}

	nodeNames := []string{}
	for k := range clients {
		nodeNames = append(nodeNames, k)
	}
	sort.Strings(nodeNames)
	nodeCount := int(math.Round(float64(len(clients)*o.UploadNodesPercentage) / 100))
	rnd := random.PseudoGenerator(o.Seed)
	picked, _ := randomPick(rnd, nodeNames, nodeCount)

	rnds := random.PseudoGenerators(rnd.Int63(), nodeCount)

	uGroup := new(errgroup.Group)
	uSemaphore := make(chan struct{}, buffer)
	for i, p := range picked {
		o := o
		i := i
		p := p
		n := clients[p]

		uSemaphore <- struct{}{}
		uGroup.Go(func() error {
			defer func() {
				<-uSemaphore
			}()

			overlay, err := n.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", p, err)
			}

			for j := 0; j < o.FilesPerNode; j++ {
				file := bee.NewRandomFile(rnds[i], "filename", o.FileSize)

				retryCount := 0
				for {
					retryCount++
					if retryCount > o.Retries {
						return fmt.Errorf("file %s upload to node %s exceeded number of retires", file.Address().String(), overlay)
					}
					time.Sleep(o.RetryDelay)

					if err := n.UploadFile(ctx, &file, false); err != nil {
						fmt.Printf("uploading file %s to node %s: %v\n", file.Address().String(), overlay, err)
						continue
					}
					break
				}

				fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlay)
			}

			return nil
		})
	}

	if err := uGroup.Wait(); err != nil {
		return err
	}

	fmt.Println("upload stress check completed successfully")
	return
}

// randomPick randomly picks n elements from the list, and returns lists of picked and unpicked elements
func randomPick(rnd *rand.Rand, list []string, n int) (picked, unpicked []string) {
	for i := 0; i < n; i++ {
		index := rnd.Intn(len(list))
		picked = append(picked, list[index])
		list = append(list[:index], list[index+1:]...)
	}
	return picked, list
}
