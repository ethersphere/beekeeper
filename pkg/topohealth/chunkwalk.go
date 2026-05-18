package topohealth

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/ethersphere/bee/v2/pkg/file/pipeline/builder"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"golang.org/x/sync/errgroup"
)

// ChunkPosition is where a chunk sits in the file's Merkle-like tree.
type ChunkPosition string

const (
	ChunkPositionRoot         ChunkPosition = "root"
	ChunkPositionIntermediate ChunkPosition = "intermediate"
	ChunkPositionLeaf         ChunkPosition = "leaf"
)

// ChunkInfo is one chunk's identity plus its position in the upload tree.
type ChunkInfo struct {
	Address  swarm.Address `json:"address"`
	Position ChunkPosition `json:"position"`
}

// SplitChunkAddresses runs the chunk-splitting pipeline locally over the same
// bytes that were (or will be) uploaded, returning the root chunk address and
// the full list of chunks bee would produce. It is deterministic for the same
// (data, redundancy level) pair, so the addresses match what bee stores.
func SplitChunkAddresses(ctx context.Context, data []byte, rLevel *redundancy.Level) (swarm.Address, []ChunkInfo, error) {
	level := redundancy.NONE
	if rLevel != nil {
		level = *rLevel
	}
	// addressOnlyCollector classifies each chunk as it is produced and discards
	// the chunk data immediately, so the splitter does not retain a copy of the
	// payload in memory for the lifetime of the walk.
	col := &addressOnlyCollector{rootPlaceholder: swarm.ZeroAddress}
	pipe := builder.NewPipelineBuilder(ctx, col, false, level)
	root, err := builder.FeedPipeline(ctx, pipe, bytes.NewReader(data))
	if err != nil {
		return swarm.ZeroAddress, nil, fmt.Errorf("split pipeline: %w", err)
	}
	// Promote whichever chunk matches the final root.
	for i := range col.chunks {
		if col.chunks[i].Address.Equal(root) {
			col.chunks[i].Position = ChunkPositionRoot
		}
	}
	return root, col.chunks, nil
}

// addressOnlyCollector classifies a chunk by its span at Put time, stores just
// the address + position, and does not keep the chunk data. The pipeline is
// single-writer so no synchronisation is needed.
type addressOnlyCollector struct {
	chunks          []ChunkInfo
	rootPlaceholder swarm.Address
}

func (c *addressOnlyCollector) Put(_ context.Context, ch swarm.Chunk) error {
	c.chunks = append(c.chunks, ChunkInfo{
		Address:  ch.Address(),
		Position: classifyBySpan(ch.Data()),
	})
	return nil
}

// classifyBySpan derives a position from the span prefix only. The final root
// promotion happens after FeedPipeline returns (we cannot know which chunk
// will be the root until the pipeline closes).
func classifyBySpan(data []byte) ChunkPosition {
	if len(data) < swarm.SpanSize {
		return ChunkPositionLeaf
	}
	span := binary.LittleEndian.Uint64(data[:swarm.SpanSize])
	if span > swarm.ChunkSize {
		return ChunkPositionIntermediate
	}
	return ChunkPositionLeaf
}

// StorerInfo is one full node's identity plus its current storage radius,
// gathered once at the start of a walk so per-chunk classification is local.
type StorerInfo struct {
	Client        *bee.Client
	Name          string
	Overlay       swarm.Address
	overlayBytes  []byte // cached for inner-loop comparison
	StorageRadius uint8
}

// GatherStorers collects overlay + storage radius for every full node in the
// cluster in parallel. Bootnodes and light nodes are excluded.
func GatherStorers(ctx context.Context, cluster orchestration.Cluster) ([]StorerInfo, error) {
	type job struct {
		client *bee.Client
		name   string
	}
	var jobs []job
	for _, n := range cluster.Nodes() {
		cfg := n.Config()
		if !cfg.FullNode || cfg.BootnodeMode {
			continue
		}
		jobs = append(jobs, job{client: n.Client(), name: n.Name()})
	}
	out := make([]StorerInfo, len(jobs))
	g, gctx := errgroup.WithContext(ctx)
	for i, j := range jobs {
		g.Go(func() error {
			addrs, err := j.client.Addresses(gctx)
			if err != nil {
				return fmt.Errorf("addresses for %s: %w", j.name, err)
			}
			st, err := j.client.Status(gctx)
			if err != nil {
				return fmt.Errorf("status for %s: %w", j.name, err)
			}
			out[i] = StorerInfo{
				Client:        j.client,
				Name:          j.name,
				Overlay:       addrs.Overlay,
				overlayBytes:  addrs.Overlay.Bytes(),
				StorageRadius: st.StorageRadius,
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func closestStorer(chunkAddr swarm.Address, storers []StorerInfo) (StorerInfo, error) {
	best := storers[0]
	for _, s := range storers[1:] {
		cmp, err := swarm.DistanceCmp(chunkAddr, s.Overlay, best.Overlay)
		if err != nil {
			return StorerInfo{}, fmt.Errorf("distance cmp: %w", err)
		}
		if cmp > 0 {
			best = s
		}
	}
	return best, nil
}

// ChunkCheck is one chunk's presence result on its intended (closest) storer.
type ChunkCheck struct {
	Address       swarm.Address `json:"address"`
	Position      ChunkPosition `json:"position"`
	StorerName    string        `json:"storer"`
	StorerOverlay swarm.Address `json:"storerOverlay"`
	Proximity     uint8         `json:"proximity"`
	StorageRadius uint8         `json:"storageRadius"`
	// OutOfAOR: PO(chunk, storer) < storer.StorageRadius. When the storer is
	// the closest in the cluster and the chunk is still out of its AOR, no
	// node covers this address (cluster-coverage gap). When Present is also
	// true, that is a direct bee#5400 bug-1 fingerprint: a node is holding a
	// chunk outside its own AOR.
	OutOfAOR bool   `json:"outOfAOR"`
	Present  bool   `json:"present"`
	Error    string `json:"error,omitempty"`
}

// PerPositionCounts is per-tree-position aggregation of a walk.
type PerPositionCounts map[ChunkPosition]int

func (c PerPositionCounts) add(p ChunkPosition) {
	c[p]++
}

// WalkResult summarises a chunk walk. Counters are exact; slices are
// truncated for logging.
type WalkResult struct {
	Checked            int               `json:"checked"`
	Missing            []ChunkCheck      `json:"missing,omitempty"`
	OutOfAORHits       []ChunkCheck      `json:"outOfAORHits,omitempty"`
	MissingTotal       PerPositionCounts `json:"missingTotal"`
	MissingOutOfAOR    PerPositionCounts `json:"missingOutOfAOR"`
	MissingInAOR       PerPositionCounts `json:"missingInAOR"`
	PresentOutOfAOR    PerPositionCounts `json:"presentOutOfAOR"`
	ProbeErrors        int               `json:"probeErrors"`
}

// WalkChunks HEADs each chunk address on its closest full node and produces
// per-chunk presence + AOR results. Concurrency is bounded by parallelism;
// the returned WalkResult holds exact per-position counters plus
// log-friendly truncated slices (maxReported per category).
func WalkChunks(ctx context.Context, storers []StorerInfo, chunks []ChunkInfo, parallelism, maxReported int) (WalkResult, error) {
	if len(storers) == 0 {
		return WalkResult{}, errors.New("no storers")
	}
	if parallelism <= 0 {
		parallelism = 32
	}

	checks := make([]ChunkCheck, len(chunks))
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
dispatch:
	for i, c := range chunks {
		select {
		case <-ctx.Done():
			break dispatch
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(idx int, ci ChunkInfo) {
			defer wg.Done()
			defer func() { <-sem }()
			storer, err := closestStorer(ci.Address, storers)
			if err != nil {
				checks[idx] = ChunkCheck{Address: ci.Address, Position: ci.Position, Error: err.Error()}
				return
			}
			po := swarm.Proximity(ci.Address.Bytes(), storer.overlayBytes)
			check := ChunkCheck{
				Address:       ci.Address,
				Position:      ci.Position,
				StorerName:    storer.Name,
				StorerOverlay: storer.Overlay,
				Proximity:     po,
				StorageRadius: storer.StorageRadius,
				OutOfAOR:      po < storer.StorageRadius,
			}
			has, herr := storer.Client.LocalHasChunk(ctx, ci.Address)
			if herr != nil {
				check.Error = herr.Error()
			} else {
				check.Present = has
			}
			checks[idx] = check
		}(i, c)
	}
	wg.Wait()

	res := WalkResult{
		MissingTotal:    PerPositionCounts{},
		MissingOutOfAOR: PerPositionCounts{},
		MissingInAOR:    PerPositionCounts{},
		PresentOutOfAOR: PerPositionCounts{},
	}
	for _, c := range checks {
		// Skip chunks we never managed to probe — counting them as missing
		// would inflate the loss metrics with transient API errors.
		if c.Error != "" {
			res.ProbeErrors++
			continue
		}
		res.Checked++
		if !c.Present {
			res.MissingTotal.add(c.Position)
			if c.OutOfAOR {
				res.MissingOutOfAOR.add(c.Position)
			} else {
				res.MissingInAOR.add(c.Position)
			}
			if len(res.Missing) < maxReported {
				res.Missing = append(res.Missing, c)
			}
		}
		if c.Present && c.OutOfAOR {
			res.PresentOutOfAOR.add(c.Position)
			if len(res.OutOfAORHits) < maxReported {
				res.OutOfAORHits = append(res.OutOfAORHits, c)
			}
		}
	}
	return res, ctx.Err()
}
