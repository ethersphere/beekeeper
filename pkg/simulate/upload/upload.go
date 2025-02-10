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
	"github.com/ethersphere/beekeeper/pkg/logging"
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
	PostageTTL           time.Duration
	PostageDepth         uint64
	PostageLabel         string
	Retries              int
	RetryDelay           time.Duration
	Seed                 int64
	Timeout              time.Duration
	UploadNodeName       string
	UploadNodePercentage int
	SyncUpload           bool
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		FileCount:            0,
		GasPrice:             "",
		MaxFileSize:          1048576, // 1mb = 1*1024*1024
		MinFileSize:          1048576, // 1mb = 1*1024*1024
		PostageTTL:           24 * time.Hour,
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
type Simulation struct {
	logger logging.Logger
}

// NewSimulation returns new upload simulation
func NewSimulation(logger logging.Logger) beekeeper.Action {
	return &Simulation{
		logger: logger,
	}
}

// Run executes upload stress
func (s *Simulation) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	s.logger.Info("running upload simulation")
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
	rnds := random.PseudoGenerators(rnd.Int63(), len(nodes))

	uGroup := new(errgroup.Group)
	uSemaphore := make(chan struct{}, concurrency)
	for i, n := range nodes {
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
				var tag api.TagResponse
				if o.SyncUpload {
					if tag, err = c.CreateTag(ctx); err != nil {
						return fmt.Errorf("create tag on node %s: %w", n, err)
					}
				}

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

					batchID, err = c.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
					if err != nil {
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							return nil
						}
						return fmt.Errorf("node %s: batch id %w", n, err)
					}

					if err := c.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID, Tag: tag.Uid}); err != nil {
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							return nil
						}
						s.logger.Infof("error: uploading file %s (size %d) to node %s, batch ID %s: %v", file.Address().String(), fileSize, overlay, batchID, err)
						continue
					}

					break
				}

				s.logger.Infof("File %s (size %d) uploaded to node %s, batch ID %s", file.Address().String(), fileSize, overlay, batchID)

				fileCount++
				if o.FileCount > 0 && fileCount >= o.FileCount {
					s.logger.Infof("Uploaded %d files to node %s", fileCount, n)
					return nil
				}

				if o.SyncUpload {
					if err = c.WaitSync(ctx, tag.Uid); err != nil {
						s.logger.Infof("sync with node %s: %v", n, err)
						continue
					}
					s.logger.Infof("file %s synced successfully with node %s", file.Address().String(), n)
				}
			}
		})
	}

	if err := uGroup.Wait(); err != nil {
		return err
	}

	s.logger.Info("upload stress completed successfully")
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
