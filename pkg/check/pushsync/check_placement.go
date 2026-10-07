package pushsync

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// placement outcomes of a single uploaded chunk
const (
	// upload succeeded and at least one node holds the chunk within its AOR
	outcomeStoredInAOR = "stored_in_aor"
	// upload succeeded but no node holds the chunk within its AOR
	outcomeFalseSuccess = "false_success"
	// upload failed or timed out and no node holds the chunk within its AOR
	outcomeReportedFailure = "reported_failure"
	// upload failed or timed out although a node holds the chunk within its AOR
	outcomeFailedButStored = "failed_but_stored"
)

// errNotSynced marks a deferred upload whose tag does not count the chunk as synced.
var errNotSynced = errors.New("not synced")

// bee counters summed over all nodes and reported as a delta over the run
var placementBeeMetrics = []string{
	"bee_pushsync_storer",
	"bee_pushsync_shallow_receipt",
	"bee_pushsync_out_of_depth_storing",
	"bee_pusher_total_could_not_sync",
}

type placementResult struct {
	chunk       swarm.Address
	uploader    string
	populated   bool // some storer node has the chunk within its AOR
	uploadErr   error
	uploadTime  time.Duration
	inAOR       []string // holders with the chunk within their AOR
	outOfAOR    []string // holders with the chunk outside their AOR
	queued      bool     // deferred: not synced and still in the uploader's queue
	outcomeName string
}

// checkPlacement uploads chunks into every neighborhood of the configured
// storage radius and then asks every node, with a HEAD request that never
// touches the network, whether it holds the chunk. A chunk only counts as
// stored when a holder has it within its area of responsibility. The check
// fails when an upload is reported as successful while no such holder exists,
// or when a chunk whose neighborhood is populated does not end up there.
//
// A direct upload reports its outcome with the response of the upload request.
// A deferred upload reports it with its tag: it is successful once the tag
// counts the chunk as synced.
func checkPlacement(ctx context.Context, c orchestration.Cluster, o Options, l logging.Logger) error {
	l.Infof("running pushsync (placement mode), storage radius %d, deferred %t", o.StorageRadius, o.Deferred)
	l.Infof("seed: %d", o.Seed)

	if o.StorageRadius > 8 {
		return fmt.Errorf("storage radius %d is too high for the placement check", o.StorageRadius)
	}

	overlays, err := c.FlattenOverlays(ctx)
	if err != nil {
		return err
	}
	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}

	storers := c.FullNodeNames()
	sort.Strings(storers)
	if o.UploadNodeCount > len(storers) {
		return fmt.Errorf("not enough full nodes in the cluster, required %d, available %d", o.UploadNodeCount, len(storers))
	}
	isStorer := make(map[string]bool, len(storers))
	for _, name := range storers {
		isStorer[name] = true
	}

	nodeNames := make([]string, 0, len(overlays))
	for name := range overlays {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	for _, name := range nodeNames {
		status, err := clients[name].Status(ctx)
		if err != nil {
			return fmt.Errorf("node %s: status: %w", name, err)
		}
		l.Infof("node %s: overlay %s neighborhood %s mode %s reported radius %d storer %t",
			name, overlays[name], neighborhood(overlays[name], o.StorageRadius), status.BeeMode, status.StorageRadius, isStorer[name])
	}

	// holdsInAOR tells whether the node is responsible for storing the chunk
	holdsInAOR := func(name string, chunk swarm.Address) bool {
		return isStorer[name] && swarm.Proximity(overlays[name].Bytes(), chunk.Bytes()) >= o.StorageRadius
	}

	before, err := sumBeeMetrics(ctx, clients, nodeNames)
	if err != nil {
		return err
	}

	rnd := random.PseudoGenerator(o.Seed)
	var results []placementResult

	for i := range o.UploadNodeCount {
		uploaderName := storers[i]
		uploader := clients[uploaderName]

		batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", uploaderName, err)
		}
		l.Infof("node %s: batch id %s", uploaderName, batchID)

		for prefix := range 1 << o.StorageRadius {
			for range o.ChunksPerNode {
				chunk, err := randomChunkInNeighborhood(rnd, l, prefix, o.StorageRadius)
				if err != nil {
					return fmt.Errorf("node %s: %w", uploaderName, err)
				}

				res := placementResult{chunk: chunk.Address(), uploader: uploaderName}
				for _, name := range storers {
					if holdsInAOR(name, res.chunk) {
						res.populated = true
						break
					}
				}

				uploadOpts := api.UploadOptions{BatchID: batchID, Direct: !o.Deferred}
				if o.Deferred {
					tag, err := uploader.CreateTag(ctx)
					if err != nil {
						return fmt.Errorf("node %s: %w", uploaderName, err)
					}
					uploadOpts.Tag = tag.Uid
				}

				uploadCtx, cancel := context.WithTimeout(ctx, o.UploadTimeout)
				t0 := time.Now()
				_, res.uploadErr = uploader.UploadChunk(uploadCtx, chunk.Data(), uploadOpts)
				res.uploadTime = time.Since(t0)
				cancel()
				if ctx.Err() != nil {
					return ctx.Err()
				}

				// give the nodes time to sync and replicate before looking for the chunk
				time.Sleep(o.RetryDelay)

				if o.Deferred && res.uploadErr == nil {
					tag, err := uploader.GetTag(ctx, uploadOpts.Tag)
					if err != nil {
						return fmt.Errorf("node %s: %w", uploaderName, err)
					}
					if tag.Synced == 0 {
						res.uploadErr = errNotSynced
					}
				}

				for _, name := range nodeNames {
					has, err := clients[name].HasChunkLocal(ctx, res.chunk)
					if err != nil {
						return fmt.Errorf("node %s: has chunk %s: %w", name, res.chunk, err)
					}
					if !has {
						continue
					}
					switch {
					case holdsInAOR(name, res.chunk):
						res.inAOR = append(res.inAOR, name)
					case name == uploaderName && errors.Is(res.uploadErr, errNotSynced):
						// the chunk waits in the upload queue, it is not stored
						res.queued = true
					default:
						res.outOfAOR = append(res.outOfAOR, name)
					}
				}

				switch {
				case res.uploadErr == nil && len(res.inAOR) > 0:
					res.outcomeName = outcomeStoredInAOR
				case res.uploadErr == nil:
					res.outcomeName = outcomeFalseSuccess
				case len(res.inAOR) > 0:
					res.outcomeName = outcomeFailedButStored
				default:
					res.outcomeName = outcomeReportedFailure
				}

				l.Infof("chunk %s neighborhood %s populated %t uploader %s upload %s in %s: %s, in AOR %v, out of AOR %v, queued %t",
					res.chunk, neighborhood(res.chunk, o.StorageRadius), res.populated, uploaderName,
					uploadStatus(res.uploadErr), res.uploadTime.Round(time.Millisecond), res.outcomeName, res.inAOR, res.outOfAOR, res.queued)

				results = append(results, res)
			}
		}
	}

	after, err := sumBeeMetrics(ctx, clients, nodeNames)
	if err != nil {
		return err
	}

	return reportPlacement(l, results, before, after)
}

// reportPlacement logs the summary and returns an error when the placement
// guarantees do not hold.
func reportPlacement(l logging.Logger, results []placementResult, before, after map[string]float64) error {
	type key struct {
		populated bool
		outcome   string
	}
	counts := make(map[key]int)
	total := map[bool]int{}
	strayCopies, queued := 0, 0
	for _, r := range results {
		counts[key{r.populated, r.outcomeName}]++
		total[r.populated]++
		if len(r.outOfAOR) > 0 {
			strayCopies++
		}
		if r.queued {
			queued++
		}
	}

	l.Info("placement summary")
	for _, populated := range []bool{true, false} {
		label := "populated neighborhood"
		if !populated {
			label = "empty neighborhood"
		}
		l.Infof("  %s: %d chunks", label, total[populated])
		for _, outcome := range []string{outcomeStoredInAOR, outcomeFalseSuccess, outcomeReportedFailure, outcomeFailedButStored} {
			l.Infof("    %-18s %d", outcome, counts[key{populated, outcome}])
		}
	}
	l.Infof("  chunks with a copy outside of the holder's AOR: %d", strayCopies)
	l.Infof("  chunks not synced and still in the uploader's queue: %d", queued)

	l.Info("bee metrics delta, summed over all nodes")
	for _, name := range placementBeeMetrics {
		l.Infof("  %-36s %.0f", name, after[name]-before[name])
	}

	falseSuccess := counts[key{true, outcomeFalseSuccess}] + counts[key{false, outcomeFalseSuccess}]
	lostInPopulated := total[true] - counts[key{true, outcomeStoredInAOR}]

	var errs []error
	if falseSuccess > 0 {
		errs = append(errs, fmt.Errorf("%d of %d uploads reported success with no node holding the chunk within its AOR", falseSuccess, len(results)))
	}
	if lostInPopulated > 0 {
		errs = append(errs, fmt.Errorf("%d of %d chunks with a populated neighborhood were not stored there", lostInPopulated, total[true]))
	}

	return errors.Join(errs...)
}

// randomChunkInNeighborhood generates a random chunk whose address starts
// with the given prefix of radius bits.
func randomChunkInNeighborhood(rnd *rand.Rand, l logging.Logger, prefix int, radius uint8) (bee.Chunk, error) {
	for {
		chunk, err := bee.NewRandomChunk(rnd, l)
		if err != nil {
			return bee.Chunk{}, err
		}
		if addressPrefix(chunk.Address(), radius) == prefix {
			return chunk, nil
		}
	}
}

// addressPrefix returns the first radius bits of the address, radius <= 8.
func addressPrefix(a swarm.Address, radius uint8) int {
	return int(a.Bytes()[0] >> (8 - radius))
}

// neighborhood returns the first radius bits of the address in binary.
func neighborhood(a swarm.Address, radius uint8) string {
	if radius == 0 {
		return "-"
	}
	return fmt.Sprintf("%0*b", int(radius), addressPrefix(a, radius))
}

func uploadStatus(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, errNotSynced):
		return "not synced"
	default:
		return fmt.Sprintf("failed (%v)", err)
	}
}

// sumBeeMetrics sums the observed bee counters over the given nodes.
func sumBeeMetrics(ctx context.Context, clients map[string]*bee.Client, nodeNames []string) (map[string]float64, error) {
	sums := make(map[string]float64, len(placementBeeMetrics))
	for _, name := range nodeNames {
		text, err := clients[name].Metrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("node %s: metrics: %w", name, err)
		}
		for line := range strings.SplitSeq(text, "\n") {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			i := strings.LastIndexByte(line, ' ')
			if i < 0 {
				continue
			}
			metric, _, _ := strings.Cut(line[:i], "{")
			for _, want := range placementBeeMetrics {
				if metric != want {
					continue
				}
				v, err := strconv.ParseFloat(line[i+1:], 64)
				if err != nil {
					return nil, fmt.Errorf("node %s: metric %s: %w", name, metric, err)
				}
				sums[metric] += v
			}
		}
	}
	return sums, nil
}
