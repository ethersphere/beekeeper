package datadurability

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type Options struct {
	Ref         string
	Concurrency int
	MaxAttempts int
}

func NewDefaultOptions() Options {
	return Options{
		Concurrency: 10,
		MaxAttempts: 10,
	}
}

var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger:  logger,
		metrics: newMetrics("check_data_durability"),
	}
}

// Run runs the check
// It downloads a file that contains a list of chunks and then attempts to download each chunk in the file.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) error {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if opts.Ref == "" {
		return fmt.Errorf("reference is required")
	}

	ref, err := swarm.ParseHexAddress(opts.Ref)
	if err != nil {
		return fmt.Errorf("parse hex ref %s: %w", opts.Ref, err)
	}

	d, err := fetchFile(ctx, c.logger, ref, cluster, opts.MaxAttempts)
	if err != nil {
		return fmt.Errorf("fetch file: %w", err)
	}

	refs, err := parseFile(bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}
	rootRef, chunkRefs := refs[0], refs[1:]

	nodes, err := findRandomEligibleNodes(ctx, rootRef, cluster)
	if err != nil {
		return fmt.Errorf("get random node: %w", err)
	}

	once := sync.Once{}
	fileStart := time.Now()
	c.metrics.ChunksCount.Set(float64(len(chunkRefs)))

	var wg sync.WaitGroup
	wg.Add(len(chunkRefs))
	limitCh := make(chan struct{}, opts.Concurrency)
	var fileAttemptCounted bool

	for i, ref := range chunkRefs {
		node := nodes[i%len(nodes)] // distribute evenly
		limitCh <- struct{}{}

		go func(i int, ref swarm.Address, node orchestration.Node) {
			defer func() {
				<-limitCh
				wg.Done()
			}()
			c.metrics.ChunkDownloadAttempts.Inc()
			cache := false
			chunkStart := time.Now()
			d, err = node.Client().DownloadChunk(ctx, ref, "", &api.DownloadOptions{Cache: &cache})
			if err != nil {
				c.logger.Errorf("download failed. %s (%d of %d). chunk=%s node=%s err=%v", percentage(i, len(chunkRefs)), i, len(chunkRefs), ref, node.Name(), err)
				c.metrics.ChunkDownloadErrors.Inc()
				once.Do(func() {
					c.metrics.FileDownloadAttempts.Inc()
					fileAttemptCounted = true
					c.metrics.FileDownloadErrors.Inc()
				})
				return
			}
			dur := time.Since(chunkStart)
			c.metrics.ChunkDownloadDuration.Observe(dur.Seconds())
			c.metrics.FileSize.Add(float64(len(d)))
		}(i, ref, node)
	}

	wg.Wait()
	if !fileAttemptCounted {
		c.metrics.FileDownloadAttempts.Inc()
	}
	dur := time.Since(fileStart)
	c.logger.Infof("done. dur=%v", dur)
	c.metrics.FileDownloadDuration.Observe(dur.Seconds())
	// wait for metrics to be pushed
	c.logger.Infof("waiting 1 minute for metrics to be pushed")
	time.Sleep(1 * time.Minute)
	return nil
}

func percentage(a, b int) string {
	return fmt.Sprintf("%.2f%%", float64(a)/float64(b)*100)
}

func fetchFile(ctx context.Context, logger logging.Logger, ref swarm.Address, cluster orchestration.Cluster, maxAttempts int) ([]byte, error) {
	logger.Infof("fetching file. ref=%s", ref.String())
	var nodes []orchestration.Node
	for _, node := range cluster.Nodes() {
		nodes = append(nodes, node)
	}

	for i := 0; i < maxAttempts; i++ {
		node := nodes[i%len(nodes)]
		d, err := node.Client().DownloadFileBytes(ctx, ref, nil)
		if err != nil {
			logger.Errorf("node: %s failed to fetch file: %v", node.Name(), err)
			continue
		}
		return d, nil

	}
	return nil, fmt.Errorf("failed to fetch file after %d attempts", maxAttempts)
}

// parseFile returns the list of references in the reader.
// It expects a list of swarm hashes where the 1st line is the root reference
// and the following lines are the individual chunk references.§
func parseFile(r io.Reader) ([]swarm.Address, error) {
	var refs []swarm.Address
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		ref, err := swarm.ParseHexAddress(line)
		if err != nil {
			return nil, fmt.Errorf("parse hex ref %s: %w", line, err)
		}
		refs = append(refs, ref)
	}

	if len(refs) < 2 {
		return nil, fmt.Errorf("invalid file format. Expected at least 1 line")
	}
	return refs, nil
}

// findRandomEligibleNodes finds nodes where the root ref is not pinned.
func findRandomEligibleNodes(ctx context.Context, rootRef swarm.Address, cluster orchestration.Cluster) ([]orchestration.Node, error) {
	nodes := cluster.Nodes()
	var eligible []orchestration.Node
	for _, node := range nodes {
		pins, err := node.Client().GetPins(ctx)
		if err != nil {
			return nil, fmt.Errorf("get pins. node=%s, err=%w", node.Name(), err)
		}
		found := slices.ContainsFunc(pins, func(ref swarm.Address) bool {
			return ref.Equal(rootRef)
		})
		if !found {
			eligible = append(eligible, node)
		}
	}

	if len(eligible) == 0 {
		return nil, fmt.Errorf("no eligible node found")
	}
	return eligible, nil
}
