package erasurecoding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/ethersphere/bee/v2/pkg/bmt"
	"github.com/ethersphere/bee/v2/pkg/bmtpool"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/keccak"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	RLevels               []*redundancy.Level
	DataSize              int64
	BulkDataSize          int64
	UploadConcurrency     int
	ChunkRetrievalTimeout time.Duration
	UseSIMD               bool
	PostageDepth          uint64
	PostageLabel          string
	PostageTTL            time.Duration
	Seed                  int64
}

func NewDefaultOptions() Options {
	return Options{
		DataSize:              307200,
		BulkDataSize:          8 << 20,
		UploadConcurrency:     16,
		ChunkRetrievalTimeout: 5 * time.Second,
		UseSIMD:               true,
		PostageDepth:          22,
		PostageLabel:          "test-label",
		PostageTTL:            24 * time.Hour,
		Seed:                  random.Int64(),
	}
}

var _ beekeeper.Action = (*Check)(nil)

type Check struct {
	logger logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

type scenario struct {
	name      string
	size      int64
	recover   bool
	everyLeaf bool
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o any) error {
	opts, ok := o.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if opts.UploadConcurrency < 1 {
		return fmt.Errorf("upload concurrency must be positive, got %d", opts.UploadConcurrency)
	}

	levels, err := resolveLevels(opts.RLevels)
	if err != nil {
		return err
	}

	if opts.UseSIMD {
		c.logger.Infof("simd hasher active: %t (batch width %d)", enableSIMD(), keccak.BatchWidth())
	}
	c.logger.Infof("seed: %d", opts.Seed)

	rnd := random.PseudoGenerator(opts.Seed)
	clients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}
	if len(clients) < 2 {
		return fmt.Errorf("need at least 2 full nodes, got %d", len(clients))
	}
	uploader, downloader := clients[0], clients[1]
	c.logger.Infof("uploader: %s, downloader: %s", uploader.Name(), downloader.Name())

	batchID, err := uploader.GetOrCreateMutableBatch(ctx, opts.PostageTTL, opts.PostageDepth, opts.PostageLabel)
	if err != nil {
		return fmt.Errorf("get or create batch: %w", err)
	}

	for _, level := range levels {
		for _, s := range scenarios(opts) {
			if err := c.runScenario(ctx, uploader, downloader, batchID, level, s, rnd, opts); err != nil {
				return fmt.Errorf("level %d %s: %w", level, s.name, err)
			}
		}
	}

	return nil
}

func resolveLevels(levels []*redundancy.Level) ([]redundancy.Level, error) {
	if len(levels) == 0 {
		return []redundancy.Level{redundancy.MEDIUM, redundancy.STRONG, redundancy.INSANE, redundancy.PARANOID}, nil
	}
	out := make([]redundancy.Level, 0, len(levels))
	for _, l := range levels {
		if l == nil || *l == redundancy.NONE || !l.Validate() {
			return nil, fmt.Errorf("invalid redundancy level %v: must be between %d and %d", l, redundancy.MEDIUM, redundancy.PARANOID)
		}
		out = append(out, *l)
	}
	return out, nil
}

func scenarios(opts Options) []scenario {
	s := []scenario{
		{name: "recover-at-limit", size: opts.DataSize, recover: true},
		{name: "fail-past-limit", size: opts.DataSize},
	}
	if opts.BulkDataSize > 0 {
		s = append(s, scenario{name: "bulk-every-leaf-at-limit", size: opts.BulkDataSize, recover: true, everyLeaf: true})
	}
	return s
}

func (c *Check) runScenario(ctx context.Context, uploader, downloader *bee.Client, batchID string, level redundancy.Level, s scenario, rnd *rand.Rand, opts Options) error {
	data := make([]byte, s.size)
	if _, err := rnd.Read(data); err != nil {
		return fmt.Errorf("generate data: %w", err)
	}

	start := time.Now()
	t, err := split(ctx, data, level)
	if err != nil {
		return err
	}
	splitDur := time.Since(start)

	dropped, err := chooseDrops(t, s, rnd)
	if err != nil {
		return err
	}
	kept := t.Store.Without(slices.Concat(dropped, t.Replicas)).Chunks()

	start = time.Now()
	if err := uploadChunks(ctx, uploader, kept, batchID, opts.UploadConcurrency); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	uploadDur := time.Since(start)

	cache := false
	fallback := true
	start = time.Now()
	got, err := downloader.DownloadBytes(ctx, t.Root, &api.DownloadOptions{
		Cache:                  &cache,
		RLevel:                 &level,
		RedundancyFallbackMode: &fallback,
		ChunkRetrievalTimeout:  &opts.ChunkRetrievalTimeout,
	})
	downloadDur := time.Since(start)

	c.logger.Infof("level %d %s: root %s, %d bytes, %d chunks uploaded, %d dropped, split %s (%.1f MB/s), upload %s (%.0f chunks/s), download %s",
		level, s.name, t.Root, len(data), len(kept), len(dropped),
		splitDur, float64(len(data))/1e6/splitDur.Seconds(),
		uploadDur, float64(len(kept))/uploadDur.Seconds(),
		downloadDur)

	if s.recover {
		if err != nil {
			return fmt.Errorf("download with %d chunks dropped: %w", len(dropped), err)
		}
		if !bytes.Equal(got, data) {
			return fmt.Errorf("downloaded data differs: got %d bytes, want %d", len(got), len(data))
		}
		return nil
	}

	if err == nil {
		if bytes.Equal(got, data) {
			return fmt.Errorf("data recovered with %d chunks dropped, one past the parity limit", len(dropped))
		}
		return fmt.Errorf("download returned %d of %d bytes without an error", len(got), len(data))
	}
	if !isRecoveryFailure(err) {
		return fmt.Errorf("download failed for an unexpected reason: %w", err)
	}
	c.logger.Infof("level %d %s: download failed as expected: %v", level, s.name, err)
	return nil
}

func chooseDrops(t *tree, s scenario, rnd *rand.Rand) ([]swarm.Address, error) {
	leaves := t.LeafNodes()
	if len(leaves) == 0 {
		return nil, errors.New("tree has no leaf intermediate chunks")
	}

	if s.everyLeaf {
		var out []swarm.Address
		for _, n := range leaves {
			d, err := drops(n, len(n.Parities), rnd)
			if err != nil {
				return nil, err
			}
			out = append(out, d...)
		}
		return out, nil
	}

	n := leaves[rnd.Intn(len(leaves))]
	count := len(n.Parities)
	if !s.recover {
		count++
	}
	return drops(n, count, rnd)
}

func uploadChunks(ctx context.Context, client *bee.Client, chunks []swarm.Chunk, batchID string, concurrency int) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	for _, ch := range chunks {
		g.Go(func() error {
			addr, err := client.UploadChunk(ctx, ch.Data(), api.UploadOptions{BatchID: batchID, Direct: true})
			if err != nil {
				return fmt.Errorf("chunk %s: %w", ch.Address(), err)
			}
			if !addr.Equal(ch.Address()) {
				return fmt.Errorf("chunk %s stored as %s", ch.Address(), addr)
			}
			return nil
		})
	}
	return g.Wait()
}

func isRecoveryFailure(err error) bool {
	var httpErr *api.HTTPStatusError
	if errors.As(err, &httpErr) {
		return httpErr.Code == http.StatusNotFound || httpErr.Code == http.StatusInternalServerError
	}
	return errors.Is(err, io.ErrUnexpectedEOF)
}

var simdOnce sync.Once

func enableSIMD() bool {
	simdOnce.Do(func() {
		if keccak.HasSIMD() {
			bmt.SetSIMDOptIn(true)
			bmtpool.Rebuild()
		}
	})
	return bmt.SIMDOptIn() && keccak.HasSIMD()
}
