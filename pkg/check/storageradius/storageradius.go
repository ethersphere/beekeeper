package storageradius

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"golang.org/x/sync/errgroup"
)

const (
	stablePollsBeforeGivingUp = 5
)

type Options struct {
	TargetFillPercent float64       // fraction of capacity to fill (>1 overshoots)
	ReserveCapacity   int           // per-node reserve capacity
	PollInterval      time.Duration // how often to check node status
	ChunksPerUpload   int           // bytes per upload request
	MinRadiusWait     time.Duration // min time watching for radius increase
	PushersIdleWait   time.Duration // max wait for backlog to settle
	DiluteDepth       uint64        // depth to dilute to (32 is max)
	DiluteWait        time.Duration // timeout for radius decrease
	UploadWavePause   time.Duration // pause between upload waves
	UploadTimeout     time.Duration // timeout per upload request
	PostageAmount     int64
	PostageLabel      string
	PostageDepth      uint64
}

func NewDefaultOptions() Options {
	return Options{
		PollInterval:      2 * time.Second,
		TargetFillPercent: 1.2,
		ChunksPerUpload:   512,
		ReserveCapacity:   4000,
		PostageDepth:      22,
		PostageAmount:     2073600000,
		PostageLabel:      "storage-radius-check",
		MinRadiusWait:     5 * time.Minute,
		PushersIdleWait:   2 * time.Minute,
		DiluteDepth:       32,
		DiluteWait:        20 * time.Minute,
		UploadWavePause:   5 * time.Second,
		UploadTimeout:     5 * time.Minute,
	}
}

var _ beekeeper.Action = (*Check)(nil)

type Check struct {
	logger logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{logger: logger}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return errors.New("invalid options type")
	}

	startedAt := time.Now()

	if err := c.waitForWarmup(ctx, cluster, o); err != nil {
		return fmt.Errorf("wait for warmup: %w", err)
	}

	fullNodes, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("get shuffled full node clients: %w", err)
	}
	if len(fullNodes) == 0 {
		return errors.New("no full nodes available, storage-radius check requires at least one full node")
	}

	status, err := fullNodes[0].Status(ctx)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	initialStorageRadius := status.StorageRadius

	uploadPlan := newUploadPlan(o)

	c.logger.Infof("cluster: %d full nodes, sizing upload to trigger radius increase", len(fullNodes))
	c.logger.Infof("target %d chunks (%.0f%% of %d), %d chunks per upload", uploadPlan.totalChunks, o.TargetFillPercent*100, o.ReserveCapacity, uploadPlan.chunksPerUpload)

	chunksBefore, err := c.logClusterState(ctx, cluster, "before")
	if err != nil {
		return err
	}

	batches, err := c.prepareBatches(ctx, fullNodes, len(fullNodes), o)
	if err != nil {
		return err
	}

	uploadedChunks := 0
	if pending, enough := c.pipelineAlreadyFull(ctx, batches, uploadPlan, o); enough {
		c.logger.Infof("pipeline already holds %d chunks, more than the %d needed, skipping uploads",
			pending, uploadPlan.chunksNeeded(o))
	} else {
		uploadedChunks, err = c.upload(ctx, batches, uploadPlan, o)
		if err != nil {
			return err
		}
	}

	storageRadius, err := c.waitForStorageRadiusIncrease(ctx, fullNodes, o)
	if err != nil {
		return err
	}

	chunksAfter, err := c.logClusterState(ctx, cluster, "after")
	if err != nil {
		return err
	}

	c.logger.Infof("uploaded %d chunks in %s, cluster reserves hold %d",
		uploadedChunks, time.Since(startedAt).Round(time.Second), chunksAfter)

	if storageRadius == 0 {
		return c.radiusUnchangedError(chunksBefore, chunksAfter, uploadedChunks, o)
	}

	c.logger.Infof("storage radius is %d (started at %d)", storageRadius, initialStorageRadius)

	if c.reservesIsAtCapacity(ctx, fullNodes, o) {
		c.logger.Infof("reserves are at capacity, remaining backlog will be evicted on arrival, diluting now")
	} else if err := c.waitForPushersIdle(ctx, fullNodes, o); err != nil {
		return err
	}
	if err := c.dilute(ctx, fullNodes, batches, storageRadius, o); err != nil {
		return err
	}

	c.logger.Infof("storage-radius check finished in %s", time.Since(startedAt).Round(time.Second))

	return nil
}

func (c *Check) waitForWarmup(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	ticker := time.NewTicker(o.PollInterval)
	defer ticker.Stop()

	for {
		clients, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(0))
		if err != nil {
			return fmt.Errorf("get full node clients: %w", err)
		}
		if len(clients) == 0 {
			return errors.New("no full nodes available")
		}

		warmingUp := 0
		for _, client := range clients {
			status, err := client.Status(ctx)
			if err != nil {
				return fmt.Errorf("node %s: status: %w", client.Name(), err)
			}
			if status.IsWarmingUp {
				warmingUp++
			}
		}

		if warmingUp == 0 {
			c.logger.Infof("all %d full nodes finished warming up", len(clients))
			return nil
		}
		c.logger.Infof("waiting for %d/%d full nodes to finish warming up", warmingUp, len(clients))

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for full nodes to finish warming up: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// uploadPlan is the computed upload sizing for a cluster.
type uploadPlan struct {
	totalChunks     int
	chunksPerUpload int
}

// newUploadPlan calculates upload size needed to trigger a radius increase.
// We size by reserve capacity with target fill percent; no need to multiply by
// neighborhoods since we're just overflowing reserves to trigger eviction/radius logic.
func newUploadPlan(o Options) uploadPlan {
	totalChunks := int(o.TargetFillPercent * float64(o.ReserveCapacity))

	return uploadPlan{
		totalChunks:     totalChunks,
		chunksPerUpload: min(o.ChunksPerUpload, totalChunks),
	}
}

// logClusterState reports each node's radius and reserve size, returning the
// cluster-wide chunk total.
func (c *Check) logClusterState(ctx context.Context, cluster orchestration.Cluster, label string) (int, error) {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return 0, fmt.Errorf("get nodes clients: %w", err)
	}

	reserveTotal := 0
	c.logger.Infof("cluster state (%s):", label)
	for name, client := range clients {
		status, err := client.Status(ctx)
		if err != nil {
			c.logger.Infof("   %s: status unavailable: %v", name, err)
			continue
		}
		reserveTotal += int(status.ReserveSize)
		c.logger.Infof("   %s: radius %d, reserve %d (within radius %d), committed depth %d",
			name, status.StorageRadius, status.ReserveSize, status.ReserveSizeWithinRadius, status.CommittedDepth)
	}

	return reserveTotal, nil
}

// chunksNeeded is how many chunks must reach a single reserve to fill it.
func (p uploadPlan) chunksNeeded(options Options) int {
	return int(float64(options.ReserveCapacity) * options.TargetFillPercent)
}

// nodeBatch pairs a postage batch with the node that owns it.
type nodeBatch struct {
	batchID string
	node    *bee.Client
}

// prepareBatches buys postage batches for each node, reusing existing ones to save time and tokens.
// Uses WaitGroup instead of errgroup to avoid hanging 900s if a batch purchase reverts on-chain.
func (c *Check) prepareBatches(ctx context.Context, nodes orchestration.ClientList, batchCount int, o Options) ([]nodeBatch, error) {
	c.logger.Infof("preparing %d postage batches in parallel (depth %d, amount %d)",
		batchCount, o.PostageDepth, o.PostageAmount)

	startedAt := time.Now()

	var (
		mutex       sync.Mutex
		batches     []nodeBatch
		failedNodes []string
		waitGroup   sync.WaitGroup
	)

	for i := range batchCount {
		node := nodes[i]
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			batchID, reused, err := c.batchForNode(ctx, node, o)

			mutex.Lock()
			defer mutex.Unlock()
			if err != nil {
				c.logger.Infof("%s: no usable batch: %v", node.Name(), err)
				failedNodes = append(failedNodes, node.Name())
				return
			}
			if reused {
				c.logger.Infof("%s: reusing batch %s", node.Name(), batchID)
			} else {
				c.logger.Infof("%s: bought batch %s", node.Name(), batchID)
			}
			batches = append(batches, nodeBatch{batchID: batchID, node: node})
		}()
	}
	waitGroup.Wait()

	if len(batches) == 0 {
		return nil, fmt.Errorf("no usable postage batches: all %d nodes failed", batchCount)
	}
	if len(failedNodes) > 0 {
		c.logger.Infof("continuing with %d/%d batches, failed on %v", len(batches), batchCount, failedNodes)
	}
	c.logger.Infof("%d batches ready in %s", len(batches), time.Since(startedAt).Round(time.Second))

	return batches, nil
}

// batchForNode finds an existing usable batch or creates a new one.
func (c *Check) batchForNode(ctx context.Context, node *bee.Client, o Options) (batchID string, reused bool, err error) {
	label := fmt.Sprintf("%s-%s", o.PostageLabel, node.Name())

	existingBatches, err := node.PostageBatches(ctx)
	if err != nil {
		return "", false, fmt.Errorf("list batches: %w", err)
	}

	for _, batch := range existingBatches {
		if !batch.Exists || batch.ImmutableFlag || !batch.Usable || batch.Label != label {
			continue
		}
		if batch.BatchTTL == 0 {
			continue // expired
		}
		if batch.Utilization >= 1<<(batch.Depth-batch.BucketDepth) {
			continue // buckets full, cannot issue more stamps
		}
		return batch.BatchID, true, nil
	}

	batchID, err = node.CreatePostageBatch(ctx, o.PostageAmount, o.PostageDepth, label, false)
	if err != nil {
		return "", false, err
	}
	return batchID, false, nil
}

// waitForStorageRadiusIncrease waits for any node's radius to climb above zero.
// Requires both stable reserves AND MinRadiusWait elapsed, since uploads are deferred.
func (c *Check) waitForStorageRadiusIncrease(ctx context.Context, nodes orchestration.ClientList, o Options) (uint8, error) {
	ticker := time.NewTicker(o.PollInterval)
	defer ticker.Stop()

	c.logger.Infof("waiting up to %s for the pushers to fill the reserves and the radius to rise", o.MinRadiusWait)

	startedAt := time.Now()
	prevReserveSize, stablePolls := -1, 0

	for {
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("timed out waiting for the storage radius to rise above 0: %w", ctx.Err())
		case <-ticker.C:
		}

		reserveTotal, pendingChunks, highestRadius := c.pipelineState(ctx, nodes)
		if highestRadius > 0 {
			c.logger.Infof("storage radius is %d after %s (reserves at %d chunks)",
				highestRadius, time.Since(startedAt).Round(time.Second), reserveTotal)
			return highestRadius, nil
		}

		elapsed := time.Since(startedAt)
		if reserveTotal == prevReserveSize {
			stablePolls++
			if stablePolls >= stablePollsBeforeGivingUp && elapsed >= o.MinRadiusWait {
				c.logger.Infof("reserves settled at %d chunks and radius still 0 after %s",
					reserveTotal, elapsed.Round(time.Second))
				return 0, nil
			}
		} else {
			if prevReserveSize >= 0 {
				c.logger.Infof("reserves at %d chunks (+%d), %d pending in the pushers, radius 0 (%s elapsed)",
					reserveTotal, reserveTotal-prevReserveSize, pendingChunks, elapsed.Round(time.Second))
			}
			stablePolls = 0
		}
		prevReserveSize = reserveTotal
	}
}

// pipelineAlreadyFull checks if reserves and pusher backlog already hold enough chunks.
func (c *Check) pipelineAlreadyFull(ctx context.Context, batches []nodeBatch, plan uploadPlan, options Options) (chunks int, full bool) {
	chunksNeeded := plan.chunksNeeded(options)

	for _, batch := range batches {
		status, err := batch.node.Status(ctx)
		if err != nil {
			continue
		}
		// A radius already above zero means bee has reacted; nothing to add.
		if status.StorageRadius > 0 {
			return int(status.ReserveSize), true
		}

		inPipeline := int(status.ReserveSize)
		if debugStore, err := batch.node.API().DebugStore.GetDebugStore(ctx); err == nil {
			inPipeline += debugStore.Upload.PendingUpload
		}
		chunks = max(chunks, inPipeline)
	}

	return chunks, chunks >= chunksNeeded
}

// upload sends random data in parallel, stopping when pipeline holds enough chunks.
func (c *Check) upload(ctx context.Context, batches []nodeBatch, plan uploadPlan, options Options) (int, error) {
	totalUploads := plan.uploadCount()

	c.logger.Infof("uploading %d chunks in %d requests across %d nodes",
		plan.totalChunks, totalUploads, len(batches))

	var (
		mutex          sync.Mutex
		uploadedChunks int
		completedCount int
	)

	enough := make(chan struct{})
	var stopOnce sync.Once
	stopUploading := func() { stopOnce.Do(func() { close(enough) }) }

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(len(batches))

	watchCtx, cancelWatch := context.WithCancel(groupCtx)
	defer cancelWatch()
	go c.stopWhenPipelineFull(watchCtx, batches, plan, options, stopUploading)

	for i := range totalUploads {
		if i > 0 && i%len(batches) == 0 {
			// Pause between waves so the watcher can poll and detect when to stop.
			// Without this, uploads finish before the watcher sees them.
			select {
			case <-enough:
			case <-groupCtx.Done():
			case <-time.After(options.UploadWavePause):
			}
		}

		batch := batches[i%len(batches)]
		group.Go(func() error {
			select {
			case <-enough:
				return nil
			default:
			}

			data := make([]byte, int64(plan.chunksPerUpload)*bee.MaxChunkSize)
			if _, err := crand.Read(data); err != nil {
				return fmt.Errorf("generate random data: %w", err)
			}

			select {
			case <-enough:
				return nil
			default:
			}

			uploadCtx, cancel := context.WithTimeout(groupCtx, options.UploadTimeout)
			address, err := batch.node.UploadBytes(uploadCtx, data, api.UploadOptions{BatchID: batch.batchID})
			cancel()
			if err != nil {
				return fmt.Errorf("upload to %s: %w", batch.node.Name(), err)
			}

			mutex.Lock()
			uploadedChunks += plan.chunksPerUpload
			completedCount++
			completed, chunks := completedCount, uploadedChunks
			mutex.Unlock()

			c.logger.Infof("upload %d/%d to %s: %s (%d/%d chunks)",
				completed, totalUploads, batch.node.Name(), address, chunks, plan.totalChunks)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return uploadedChunks, err
	}

	return uploadedChunks, nil
}

// radiusUnchangedError explains why the radius stayed at zero.
func (c *Check) radiusUnchangedError(chunksBefore, chunksAfter, uploadedChunks int, options Options) error {
	if chunksAfter <= chunksBefore {
		return fmt.Errorf("storage radius is still 0 and the reserves did not grow (%d chunks before, %d after, %d uploaded): "+
			"bee accepted the uploads but the pushers are not delivering them",
			chunksBefore, chunksAfter, uploadedChunks)
	}
	return fmt.Errorf("storage radius is still 0: reserves grew from %d to %d chunks, "+
		"but no node exceeded its %d-chunk capacity long enough to force an increase",
		chunksBefore, chunksAfter, options.ReserveCapacity)
}

// reservesIsAtCapacity checks if all nodes have reserves at 95% or higher.
func (c *Check) reservesIsAtCapacity(ctx context.Context, nodes orchestration.ClientList, options Options) bool {
	full := options.ReserveCapacity * 95 / 100
	sawNode := false

	for _, node := range nodes {
		status, err := node.Status(ctx)
		if err != nil {
			continue
		}
		sawNode = true
		if int(status.ReserveSize) < full {
			return false
		}
	}

	return sawNode
}

// waitForPushersIdle waits for the pusher backlog to stop shrinking.
// Does not wait for zero, as it may never empty on a repeatedly-filled cluster.
func (c *Check) waitForPushersIdle(ctx context.Context, nodes orchestration.ClientList, options Options) error {
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	c.logger.Infof("waiting up to %s for the pusher backlog to settle", options.PushersIdleWait)

	startedAt := time.Now()
	prevPendingChunks, stablePolls := -1, 0

	for {
		_, pendingChunks, _ := c.pipelineState(ctx, nodes)
		elapsed := time.Since(startedAt)

		if pendingChunks == 0 {
			c.logger.Infof("pushers idle after %s", elapsed.Round(time.Second))
			return nil
		}

		if pendingChunks >= prevPendingChunks && prevPendingChunks >= 0 {
			stablePolls++
			if stablePolls >= stablePollsBeforeGivingUp {
				c.logger.Infof("pusher backlog stable at %d chunks after %s, continuing", pendingChunks, elapsed.Round(time.Second))
				return nil
			}
		} else {
			stablePolls = 0
		}

		if elapsed >= options.PushersIdleWait {
			c.logger.Infof("pusher backlog still %d chunks after %s, continuing anyway", pendingChunks, elapsed.Round(time.Second))
			return nil
		}

		c.logger.Infof("%d chunks still pending in the pushers (%s elapsed)", pendingChunks, elapsed.Round(time.Second))
		prevPendingChunks = pendingChunks

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for the pushers to drain, %d chunks still pending: %w", pendingChunks, ctx.Err())
		case <-ticker.C:
		}
	}
}

// dilute increases batch depths to push chunks outside the storage radius.
func (c *Check) dilute(ctx context.Context, nodes orchestration.ClientList, batches []nodeBatch, startRadius uint8, options Options) error {
	c.logger.Infof("diluting %d batches to depth %d to push chunks outside the storage radius",
		len(batches), options.DiluteDepth)

	dilutedCount := 0
	for _, batch := range batches {
		stamp, err := batch.node.PostageStamp(ctx, batch.batchID)
		if err != nil {
			c.logger.Infof("%s: cannot read batch %s: %v", batch.node.Name(), batch.batchID, err)
			continue
		}
		if uint64(stamp.Depth) >= options.DiluteDepth {
			c.logger.Infof("%s: batch already at depth %d, skipping", batch.node.Name(), stamp.Depth)
			continue
		}

		if err := batch.node.DilutePostageBatch(ctx, batch.batchID, options.DiluteDepth, ""); err != nil {
			c.logger.Infof("%s: dilute to depth %d failed: %v", batch.node.Name(), options.DiluteDepth, err)
			continue
		}
		dilutedCount++
		c.logger.Infof("%s: diluted batch from depth %d to %d", batch.node.Name(), stamp.Depth, options.DiluteDepth)
	}

	if dilutedCount == 0 {
		return errors.New("no batches were diluted, cannot provoke a radius decrease")
	}

	return c.waitForStorageRadiusDecrease(ctx, nodes, startRadius, options)
}

// waitForStorageRadiusDecrease waits for any node's radius to drop below the starting radius.
func (c *Check) waitForStorageRadiusDecrease(ctx context.Context, nodes orchestration.ClientList, startRadius uint8, options Options) error {
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	c.logger.Infof("waiting up to %s for any node's storage radius to fall below %d", options.DiluteWait, startRadius)

	startedAt := time.Now()
	decreaseThreshold := options.ReserveCapacity * 8 / 10

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for the storage radius to decrease: %w", ctx.Err())
		case <-ticker.C:
		}

		lowestRadius, chunksWithinRadius, pullsyncRate, reachable := c.radiusDecreaseState(ctx, nodes)
		if reachable && lowestRadius < startRadius {
			c.logger.Infof("storage radius decreased %d -> %d after %s",
				startRadius, lowestRadius, time.Since(startedAt).Round(time.Second))
			return nil
		}

		if !reachable {
			c.logger.Infof("no node answered, retrying")
			continue
		}

		elapsed := time.Since(startedAt)
		if elapsed >= options.DiluteWait {
			return fmt.Errorf("storage radius stayed at %d after %s: %d chunks within radius (threshold %d), pullsync rate %.2f",
				startRadius, options.DiluteWait, chunksWithinRadius, decreaseThreshold, pullsyncRate)
		}

		c.logger.Infof("radius still %d, %d chunks within radius (threshold %d), pullsync %.2f (%s elapsed)",
			lowestRadius, chunksWithinRadius, decreaseThreshold, pullsyncRate, elapsed.Round(time.Second))
	}
}

// pipelineState returns total reserves, pending chunks, and the highest radius in the cluster.
func (c *Check) pipelineState(ctx context.Context, nodes orchestration.ClientList) (reserveTotal, pendingChunks int, highestRadius uint8) {
	for _, node := range nodes {
		if status, err := node.Status(ctx); err == nil {
			reserveTotal += int(status.ReserveSize)
			highestRadius = max(highestRadius, status.StorageRadius)
		}
		if debugStore, err := node.API().DebugStore.GetDebugStore(ctx); err == nil {
			pendingChunks += debugStore.Upload.PendingUpload
		}
	}
	return reserveTotal, pendingChunks, highestRadius
}

// uploadCount returns the total number of upload requests needed.
func (p uploadPlan) uploadCount() int {
	return max((p.totalChunks+p.chunksPerUpload-1)/p.chunksPerUpload, 1)
}

// stopWhenPipelineFull halts uploads once the pipeline holds enough chunks or radius rises.
// Measures growth from baseline to handle pre-filled clusters.
func (c *Check) stopWhenPipelineFull(ctx context.Context, batches []nodeBatch, plan uploadPlan, options Options, stopUploading func()) {
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	chunksNeeded := plan.chunksNeeded(options)
	baseReserve, basePending := c.pipelineBaseline(ctx, batches)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		var (
			pendingChunks  int
			largestReserve int
			sawDebugStore  bool
		)

		for _, batch := range batches {
			status, err := batch.node.Status(ctx)
			if err != nil {
				continue
			}

			if status.StorageRadius > 0 {
				c.logger.Infof("%s reports storage radius %d, stopping further uploads",
					batch.node.Name(), status.StorageRadius)
				stopUploading()
				return
			}
			if int(status.ReserveSize) > options.ReserveCapacity {
				c.logger.Infof("%s reached %d/%d chunks, stopping further uploads",
					batch.node.Name(), status.ReserveSize, options.ReserveCapacity)
				stopUploading()
				return
			}
			largestReserve = max(largestReserve, int(status.ReserveSize))

			if debugStore, err := batch.node.API().DebugStore.GetDebugStore(ctx); err == nil {
				pendingChunks += debugStore.Upload.PendingUpload
				sawDebugStore = true
			}
		}

		delivered := largestReserve - baseReserve
		queued := pendingChunks - basePending
		if sawDebugStore && delivered+queued >= chunksNeeded {
			c.logger.Infof("%d chunks added to the pipeline (%d delivered, %d queued) covers the %d needed, stopping further uploads",
				delivered+queued, delivered, queued, chunksNeeded)
			stopUploading()
			return
		}
	}
}

// pipelineBaseline snapshots reserve size and pusher backlog at the start.
func (c *Check) pipelineBaseline(ctx context.Context, batches []nodeBatch) (reserve, pending int) {
	for _, batch := range batches {
		if status, err := batch.node.Status(ctx); err == nil {
			reserve = max(reserve, int(status.ReserveSize))
		}
		if debugStore, err := batch.node.API().DebugStore.GetDebugStore(ctx); err == nil {
			pending += debugStore.Upload.PendingUpload
		}
	}
	return reserve, pending
}

// radiusDecreaseState returns the lowest radius, chunks within it, and pullsync rate across the cluster.
func (c *Check) radiusDecreaseState(ctx context.Context, nodes orchestration.ClientList) (lowestRadius uint8, chunksWithinRadius int, pullsyncRate float64, reachable bool) {
	lowestRadius = uint8(31)
	for _, node := range nodes {
		status, err := node.Status(ctx)
		if err != nil {
			continue
		}
		reachable = true
		lowestRadius = min(lowestRadius, status.StorageRadius)
		chunksWithinRadius += int(status.ReserveSizeWithinRadius)
		pullsyncRate += status.PullsyncRate
	}
	return lowestRadius, chunksWithinRadius, pullsyncRate, reachable
}
