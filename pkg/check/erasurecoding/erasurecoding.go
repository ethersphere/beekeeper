package erasurecoding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"golang.org/x/sync/errgroup"
)

const uploadConcurrency = 16

// Options for the erasure-coding check.
type Options struct {
	Levels          []uint8       // redundancy levels to test: 1 medium, 2 strong, 3 insane, 4 paranoid
	DataSizes       []int64       // sizes of the random blobs to split
	AllDepths       bool          // also test the intermediate chunks above the leaf level
	PostageDepth    uint64        // depth of the postage batch
	PostageLabel    string        // label of the postage batch
	PostageTTL      time.Duration // lifetime of the postage batch
	DownloadTimeout time.Duration // time allowed for each download
	Seed            int64
}

// NewDefaultOptions returns the default options.
func NewDefaultOptions() Options {
	return Options{
		Levels:          []uint8{uint8(redundancy.MEDIUM), uint8(redundancy.STRONG), uint8(redundancy.INSANE), uint8(redundancy.PARANOID)},
		DataSizes:       []int64{256 * 1024}, // 256 KiB: one group at every redundancy level
		PostageDepth:    22,
		PostageLabel:    "erasure-coding-check",
		PostageTTL:      24 * time.Hour,
		DownloadTimeout: 60 * time.Second,
		Seed:            time.Now().UnixNano(),
	}
}

var _ beekeeper.Action = (*Check)(nil)

// Check uploads chunk trees with some chunks withheld, and checks that a download
// succeeds exactly when the loss is within what erasure coding can rebuild.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new erasure-coding check.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{logger: logger}
}

// scenario is one upload and download.
type scenario struct {
	level    redundancy.Level
	size     int64
	group    int
	c        Case
	fallback bool
}

func (s scenario) String() string {
	return fmt.Sprintf("level %d, %d bytes, group %d, %s, fallback %v", s.level, s.size, s.group, s.c, s.fallback)
}

// Run implements beekeeper.Action.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o any) error {
	opts, ok := o.(Options)
	if !ok {
		return errors.New("invalid options type")
	}

	if err := opts.validate(); err != nil {
		return err
	}

	uploader, downloader, err := clients(ctx, cluster, opts.Seed)
	if err != nil {
		return err
	}

	batchID, err := uploader.GetOrCreateMutableBatch(ctx, opts.PostageTTL, opts.PostageDepth, opts.PostageLabel)
	if err != nil {
		return fmt.Errorf("get or create batch: %w", err)
	}

	c.logger.Infof("uploading from %s, downloading from %s", uploader.Name(), downloader.Name())

	rnd := random.PseudoGenerator(opts.Seed)

	var failures []string
	total := 0

	for _, lvl := range opts.Levels {
		level := redundancy.Level(lvl)

		for _, size := range opts.DataSizes {
			// A reference tree tells how many groups this size has; every scenario then
			// splits fresh data of the same size, whose tree has the same shape.
			ref, err := Split(ctx, randomData(rnd, size), level)
			if err != nil {
				return fmt.Errorf("level %d, %d bytes: %w", level, size, err)
			}

			groups := len(ref.SelectGroups(opts.AllDepths))
			c.logger.Infof("level %d, %d bytes: %d chunks, %d groups under test", level, size, len(ref.Chunks), groups)

			for group := range groups {
				for _, cs := range Cases {
					// Fresh data for every upload: chunks left in the network or in the
					// download node's cache by one scenario must not satisfy the next.
					for _, fallback := range []bool{false, true} {
						s := scenario{level: level, size: size, group: group, c: cs, fallback: fallback}
						total++

						if err := c.run(ctx, s, uploader, downloader, batchID, rnd, opts); err != nil {
							if ctx.Err() != nil {
								return fmt.Errorf("%s: %w", s, ctx.Err())
							}
							c.logger.Errorf("FAIL %s: %v", s, err)
							failures = append(failures, fmt.Sprintf("%s: %v", s, err))

							continue
						}

						c.logger.Infof("ok   %s", s)
					}
				}
			}
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("%d of %d scenarios failed:\n  %s", len(failures), total, strings.Join(failures, "\n  "))
	}

	c.logger.Infof("all %d scenarios passed", total)

	return nil
}

// run uploads a fresh tree minus the withheld chunks and checks the download against the expectation.
func (c *Check) run(ctx context.Context, s scenario, uploader, downloader *bee.Client, batchID string, rnd *rand.Rand, opts Options) error {
	data := randomData(rnd, s.size)

	tree, err := Split(ctx, data, s.level)
	if err != nil {
		return err
	}

	groups := tree.SelectGroups(opts.AllDepths)
	if s.group >= len(groups) {
		return fmt.Errorf("tree has %d groups under test, want group %d", len(groups), s.group)
	}

	withheld := groups[s.group].Drops(s.c, rnd)

	if err := upload(ctx, uploader, batchID, tree.Chunks, withheld); err != nil {
		return err
	}

	dctx, cancel := context.WithTimeout(ctx, opts.DownloadTimeout)
	defer cancel()

	got, dlErr := downloader.DownloadBytes(dctx, tree.Root, &api.DownloadOptions{RedundancyFallbackMode: &s.fallback})
	downloaded := dlErr == nil && bytes.Equal(got, data)

	c.logger.Debugf("%s: withheld %d of %d chunks, download ok %v, err %v", s, len(withheld), len(tree.Chunks), downloaded, dlErr)

	switch want := s.c.ExpectDownload(s.fallback); {
	case want && !downloaded && dlErr != nil:
		return fmt.Errorf("withheld %d of %d chunks, expected the download to succeed: %w", len(withheld), len(tree.Chunks), dlErr)
	case want && !downloaded:
		return fmt.Errorf("withheld %d of %d chunks, download returned different data", len(withheld), len(tree.Chunks))
	case !want && downloaded:
		return fmt.Errorf("withheld %d of %d chunks, expected the download to fail but it returned the data", len(withheld), len(tree.Chunks))
	}

	return nil
}

// upload sends every chunk except the withheld ones.
func upload(ctx context.Context, client *bee.Client, batchID string, chunks []swarm.Chunk, withheld []swarm.Address) error {
	skip := make(map[string]bool, len(withheld))
	for _, a := range withheld {
		skip[a.ByteString()] = true
	}

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(uploadConcurrency)

	for _, ch := range chunks {
		if skip[ch.Address().ByteString()] {
			continue
		}

		group.Go(func() error {
			if _, err := client.UploadChunk(gctx, ch.Data(), api.UploadOptions{BatchID: batchID, Direct: true}); err != nil {
				return fmt.Errorf("upload chunk %s: %w", ch.Address(), err)
			}

			return nil
		})
	}

	return group.Wait()
}

func randomData(rnd *rand.Rand, size int64) []byte {
	data := make([]byte, size)
	_, _ = rnd.Read(data)

	return data
}

func (o Options) validate() error {
	if len(o.Levels) == 0 {
		return errors.New("no redundancy levels configured")
	}
	for _, l := range o.Levels {
		if level := redundancy.Level(l); level == redundancy.NONE || !level.Validate() {
			return fmt.Errorf("redundancy level %d must be between 1 and 4", l)
		}
	}
	if len(o.DataSizes) == 0 {
		return errors.New("no data sizes configured")
	}
	for _, s := range o.DataSizes {
		if s <= swarm.ChunkSize {
			return fmt.Errorf("data size %d must be larger than one chunk (%d bytes) to have an intermediate chunk", s, swarm.ChunkSize)
		}
	}
	if o.DownloadTimeout <= 0 {
		return errors.New("download timeout must be positive")
	}

	return nil
}

// clients picks a node to upload from and a different node to download from.
func clients(ctx context.Context, cluster orchestration.Cluster, seed int64) (uploader, downloader *bee.Client, err error) {
	nodes, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(seed))
	if err != nil {
		return nil, nil, fmt.Errorf("get shuffled full node clients: %w", err)
	}
	if len(nodes) < 2 {
		return nil, nil, errors.New("erasure-coding check needs at least two full nodes")
	}

	return nodes[0], nodes[1], nil
}
