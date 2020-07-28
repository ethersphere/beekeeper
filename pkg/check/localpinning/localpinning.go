package localpinning

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents localpinning check options
type Options struct {
	DBCapacity     int64
	FileName       string
	LargeFileCount int
	LargeFileSize  int64
	Seed           int64
	SmallFileSize  int64
}

// Check ...
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(downloadedCounter)
	pusher.Collector(downloadTimeGauge)
	pusher.Collector(downloadTimeHistogram)
	pusher.Collector(pinnedCounter)
	pusher.Collector(notPinnedCounter)
	pusher.Collector(retrievedCounter)
	pusher.Collector(notRetrievedCounter)
	pusher.Collector(unpinnedCounter)
	pusher.Collector(notUnpinnedCounter)

	pusher.Format(expfmt.FmtText)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	// upload reference file to random node
	uIndex := rnd.Intn(c.Size())
	refFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.SmallFileSize)
	t0 := time.Now()
	if err := c.Nodes[uIndex].UploadFile(ctx, &refFile); err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	d0 := time.Since(t0)
	fmt.Printf("Node %d. Reference file %s (size: %d) uploaded successfully to node %s\n", uIndex, refFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	uploadedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
	uploadTimeGauge.WithLabelValues(overlays[uIndex].String(), refFile.Address().String()).Set(d0.Seconds())
	uploadTimeHistogram.Observe(d0.Seconds())

	// download reference from random node
	dSkip := []int{uIndex}
	dIndexes, err := randomIndexes(rnd, 1, c.Size(), dSkip)
	dIndex := dIndexes[0]
	t1 := time.Now()
	refSize, refHash, err := c.Nodes[dIndex].DownloadFile(ctx, refFile.Address())
	if err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", dIndex, err)
	}
	d1 := time.Since(t1)

	downloadedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
	downloadTimeGauge.WithLabelValues(overlays[dIndex].String(), refFile.Address().String()).Set(d1.Seconds())
	downloadTimeHistogram.Observe(d1.Seconds())

	if !bytes.Equal(refFile.Hash(), refHash) {
		notRetrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("Node %d. Reference file %s (size: %d) not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d", dIndex, refFile.Address().String(), refSize, overlays[dIndex].String(), refFile.Size(), refSize)
	}
	retrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
	fmt.Printf("Node %d. Reference file %s (size: %d) downloaded successfully from node %s\n", dIndex, refFile.Address().String(), refSize, overlays[dIndex].String())

	smallFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.SmallFileSize)
	t2 := time.Now()
	if err := c.Nodes[uIndex].UploadFile(ctx, &smallFile); err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	d2 := time.Since(t2)
	fmt.Printf("Node %d. Small file %s (size: %d) uploaded successfully to node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	uploadedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
	uploadTimeGauge.WithLabelValues(overlays[uIndex].String(), smallFile.Address().String()).Set(d2.Seconds())
	uploadTimeHistogram.Observe(d2.Seconds())

	pinned, err := c.Nodes[uIndex].PinChunk(ctx, smallFile.Address())
	if err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	if !pinned {
		notPinnedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("Node %d. Small file %s (size: %d) not pinned successfully on node %s", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())
	}
	pinnedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
	fmt.Printf("Node %d. Small file %s (size: %d) pinned successfully on node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	for i := 0; i < o.LargeFileCount; i++ {
		largeFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.LargeFileSize)
		t3 := time.Now()
		if err := c.Nodes[uIndex].UploadFile(ctx, &largeFile); err != nil {
			metricsHandler(pusher, pushMetrics)
			return fmt.Errorf("node %d: %w", uIndex, err)
		}
		d3 := time.Since(t3)
		fmt.Printf("Node %d. Large file %s (size: %d) uploaded successfully to node %s\n", uIndex, largeFile.Address().String(), o.LargeFileSize, overlays[uIndex].String())

		uploadedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
		uploadTimeGauge.WithLabelValues(overlays[uIndex].String(), largeFile.Address().String()).Set(d3.Seconds())
		uploadTimeHistogram.Observe(d3.Seconds())

		t4 := time.Now()
		smallSize, smallHash, err := c.Nodes[dIndex].DownloadFile(ctx, smallFile.Address())
		if err != nil {
			metricsHandler(pusher, pushMetrics)
			return fmt.Errorf("node %d: %w", dIndex, err)
		}
		d4 := time.Since(t4)

		downloadedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
		downloadTimeGauge.WithLabelValues(overlays[dIndex].String(), smallFile.Address().String()).Set(d4.Seconds())
		downloadTimeHistogram.Observe(d4.Seconds())

		if !bytes.Equal(smallFile.Hash(), smallHash) {
			notRetrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
			metricsHandler(pusher, pushMetrics)
			return fmt.Errorf("Node %d. Small file %s (size: %d) not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d", dIndex, smallFile.Address().String(), smallSize, overlays[dIndex].String(), smallFile.Size(), smallSize)
		}

		retrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
		fmt.Printf("Node %d. Small file %s (size: %d) downloaded successfully from node %s\n", dIndex, smallFile.Address().String(), smallSize, overlays[dIndex].String())
	}

	pins, err := c.Nodes[uIndex].PinnedChunks(ctx)
	if err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	pinFound := false
	for _, pin := range pins.Chunks {
		if pin.Address.Equal(smallFile.Address()) {
			pinFound = true
		}
	}
	if !pinFound {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("Node %d. Small file %s not found in the pinned chunk list", uIndex, smallFile.Address().String())
	}

	// cleanup
	unpinned, err := c.Nodes[uIndex].UnpinChunk(ctx, smallFile.Address())
	if err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	if !unpinned {
		notUnpinnedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("Node %d. Small file %s (size: %d) not unpinned successfully on node %s", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())
	}
	unpinnedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
	fmt.Printf("Node %d. Small file %s (size: %d) unpinned successfully on node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	metricsHandler(pusher, pushMetrics)

	return
}

// randomIndexes finds n random indexes <max and but excludes skipped
func randomIndexes(rnd *rand.Rand, n, max int, skipped []int) (indexes []int, err error) {
	if n > max-len(skipped) {
		return []int{}, fmt.Errorf("not enough nodes")
	}

	found := false
	for !found {
		i := rnd.Intn(max)
		if !contains(indexes, i) && !contains(skipped, i) {
			indexes = append(indexes, i)
		}
		if len(indexes) == n {
			found = true
		}
	}

	return
}

// contains checks if a given set of int contains given int
func contains(s []int, v int) bool {
	for _, a := range s {
		if a == v {
			return true
		}
	}
	return false
}

// metricsHandler wraps pushing metrics condition
func metricsHandler(pusher *push.Pusher, yes bool) {
	if yes {
		if err := pusher.Push(); err != nil {
			fmt.Printf("push metrics: %s\n", err)
		}
	}
}
