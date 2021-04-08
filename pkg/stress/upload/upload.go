package upload

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/stress"
	"golang.org/x/sync/errgroup"
)

// compile stress whether Upload implements interface
var _ stress.Stress = (*Upload)(nil)

// UploadStress stress
type Upload struct{}

// NewUploadStress returns new ping stress
func NewUpload() *Upload {
	return &Upload{}
}

// Run executes upload stress
func (u *Upload) Run(ctx context.Context, cluster *bee.Cluster, o stress.Options) (err error) {
	concurrency := 100

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
	picked := randomPick(rnd, nodeNames, nodeCount)

	rnds := random.PseudoGenerators(rnd.Int63(), nodeCount)

	uGroup := new(errgroup.Group)
	uSemaphore := make(chan struct{}, concurrency)
	for i, p := range picked {
		i := i
		p := p
		n := clients[p]

		uSemaphore <- struct{}{}
		uGroup.Go(func() error {
			defer func() {
				<-uSemaphore
			}()

			ctx, ctxCancel := context.WithTimeout(ctx, o.Timeout)
			defer ctxCancel()

			overlay, err := n.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", p, err)
			}

			for {
				file := bee.NewRandomFile(rnds[i], "filename", o.FileSize)

				retryCount := 0
				for {
					retryCount++
					if retryCount > o.Retries {
						return fmt.Errorf("file %s upload to node %s exceeded number of retires", file.Address().String(), overlay)
					}

					select {
					case <-time.After(o.RetryDelay):
					case <-ctx.Done():
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							return nil
						}
						return ctx.Err()
					}

					// add some buffer to ensure depth is enough
					depth := 2 + bee.EstimatePostageBatchDepth(file.Size())
					batchID, err := n.CreatePostageBatch(ctx, o.PostageAmount, depth, "test-label")
					if err != nil {
						return fmt.Errorf("created batched id %w", err)
					}

					fmt.Printf("created batched id %s", batchID)

					time.Sleep(o.PostageWait)

					if err := n.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
						fmt.Printf("error: uploading file %s to node %s: %v\n", file.Address().String(), overlay, err)
						continue
					}
					break
				}

				fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlay)
			}
		})
	}

	if err := uGroup.Wait(); err != nil {
		return err
	}

	fmt.Println("upload stress completed successfully")
	return
}

// randomPick randomly picks n elements from the list, and returns lists of picked elements
func randomPick(rnd *rand.Rand, list []string, n int) (picked []string) {
	for i := 0; i < n; i++ {
		index := rnd.Intn(len(list))
		picked = append(picked, list[index])
		list = append(list[:index], list[index+1:]...)
	}
	return
}
