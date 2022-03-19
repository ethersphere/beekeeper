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
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"golang.org/x/sync/errgroup"
)

// Options represents simulation options
type Options struct {
	FileSize             int64
	FileCount            int64
	TotalSize            int64
	GasPrice             string
	PostageAmount        int64
	PostageDepth         uint64
	PostageLabel         string
	Retries              int
	RetryDelay           time.Duration
	Seed                 int64
	Timeout              time.Duration
	UploadNodeName       string
	UploadNodePercentage int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		FileSize:             1,
		FileCount:            0,
		TotalSize:            0,
		GasPrice:             "",
		PostageAmount:        1000,
		PostageDepth:         16,
		PostageLabel:         "test-label",
		Retries:              5,
		RetryDelay:           1 * time.Second,
		Seed:                 0,
		Timeout:              5 * time.Minute,
		UploadNodeName:       "",
		UploadNodePercentage: 50,
	}
}

// compile simulation whether Upload implements interface
var _ beekeeper.Action = (*Simulation)(nil)

// Simulation instance
type Simulation struct{}

// NewSimulation returns new upload simulation
func NewSimulation() beekeeper.Action {
	return &Simulation{}
}

// Run executes upload stress
func (s *Simulation) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	fmt.Println("running upload simulation")
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	fmt.Println("OPTIONS", o)

	concurrency := 100

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("node clients: %w", err)
	}

	rnd := random.PseudoGenerator(o.Seed)

	// choose nodes for upload
	nodes := []string{}
	if o.UploadNodeName != "" {
		nodes = append(nodes, o.UploadNodeName)
	} else {
		if o.UploadNodePercentage < 0 || o.UploadNodePercentage > 100 {
			return fmt.Errorf("upload-nodes-percentage must be number between 0 and 100")
		}

		allNodes := []string{}
		for k := range clients {
			allNodes = append(allNodes, k)
		}

		nodeCount := int(math.Round(float64(len(clients)*o.UploadNodePercentage) / 100))
		nodes = randomPick(rnd, allNodes, nodeCount)
	}
	sort.Strings(nodes)

	// return fmt.Errorf("dummy")
	rnds := random.PseudoGenerators(rnd.Int63(), len(nodes))

	uGroup := new(errgroup.Group)
	uSemaphore := make(chan struct{}, concurrency)
	for i, n := range nodes {
		i := i
		n := n
		c := clients[n]

		uSemaphore <- struct{}{}
		uGroup.Go(func() error {
			defer func() {
				<-uSemaphore
			}()

			ctx, ctxCancel := context.WithTimeout(ctx, o.Timeout)
			defer ctxCancel()

			overlay, err := c.Overlay(ctx)
			if err != nil {
				return fmt.Errorf("node %s: %w", n, err)
			}

			for {
				file := bee.NewRandomFile(rnds[i], "filename", o.FileSize)
				var batchID string

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

					batchID, err = c.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
					if err != nil {
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							return nil
						}
						return fmt.Errorf("node %s: batch id %w", n, err)
					}

					if err := c.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							return nil
						}
						fmt.Printf("error: uploading file %s to node %s, batch ID %s: %v\n", file.Address().String(), overlay, batchID, err)
						continue
					}

					break
				}

				fmt.Printf("File %s uploaded successfully to node %s, batch ID %s\n", file.Address().String(), overlay, batchID)
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
