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
	FileCount            int64
	GasPrice             string
	MaxFileSize          int64
	MinFileSize          int64
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
		FileCount:            0,
		GasPrice:             "",
		MaxFileSize:          1048576, // 1mb = 1*1024*1024
		MinFileSize:          1048576, // 1mb = 1*1024*1024
		PostageAmount:        1000,
		PostageDepth:         16,
		PostageLabel:         "test-label",
		Retries:              5,
		RetryDelay:           1 * time.Second,
		Seed:                 0,
		Timeout:              1 * time.Minute,
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

	if o.MinFileSize > o.MaxFileSize {
		return fmt.Errorf("file min size must be less or equal than file max size")
	}

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

	concurrency := 100
	rnds := random.PseudoGenerators(time.Now().UnixNano(), len(nodes))

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

			var fileCount int64
			for {
				// set file size
				fileSize := rnds[i].Int63n(o.MaxFileSize-o.MinFileSize+1) + o.MinFileSize
				file := bee.NewRandomFile(rnds[i], "filename", fileSize)

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
						fmt.Printf("error: uploading file %s (size %d) to node %s, batch ID %s: %v\n", file.Address().String(), fileSize, overlay, batchID, err)
						continue
					}

					break
				}

				fmt.Printf("File %s (size %d) (name %s) uploaded to node %s, batch ID %s\n", file.Address().String(), fileSize, file.Name(), n, batchID)

				fileCount++
				if o.FileCount > 0 && fileCount >= o.FileCount {
					fmt.Printf("Uploaded %d files to node %s\n", fileCount, n)
					return nil
				}
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
