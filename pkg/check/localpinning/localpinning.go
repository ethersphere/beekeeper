package localpinning

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents localpinning check options
type Options struct {
	FileName       string
	LargeFileCount int
	LargeFileSize  int64
	Seed           int64
	SmallFileSize  int64
}

// Check uploads a small file to the cluster and pins it, it then pumps large files that overflow the node's local
// storage. It then tries to download the pinned file again after all larger file uploads have finished.
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

	uIndex := rnd.Intn(c.Size())
	dSkip := []int{uIndex}
	dIndexes, err := randomIndexes(rnd, 1, c.Size(), dSkip)
	if err != nil {
		return fmt.Errorf("random indexes: %w", err)
	}
	dIndex := dIndexes[0]

	smallFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.SmallFileSize)
	t1 := time.Now()
	if err := c.Nodes[uIndex].UploadFile(ctx, &smallFile, true); err != nil {
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	d1 := time.Since(t1)
	fmt.Printf("Node %d. Small file %s (size: %d bytes) uploaded successfully to node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	uploadedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
	uploadTimeGauge.WithLabelValues(overlays[uIndex].String(), smallFile.Address().String()).Set(d1.Seconds())
	uploadTimeHistogram.Observe(d1.Seconds())

	pinnedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
	fmt.Printf("Node %d. Small file %s (size: %d bytes) pinned successfully on node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	for i := 0; i < o.LargeFileCount; i++ {
		largeFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.LargeFileSize)
		t2 := time.Now()
		// upload the large files without pinning them
		if err := c.Nodes[uIndex].UploadFile(ctx, &largeFile, false); err != nil {
			metricsHandler(pusher, pushMetrics)
			return fmt.Errorf("node %d: %w", uIndex, err)
		}
		d2 := time.Since(t2)
		fmt.Printf("Node %d. Large file %s (size: %d bytes) uploaded successfully to node %s\n", uIndex, largeFile.Address().String(), o.LargeFileSize, overlays[uIndex].String())

		uploadedCounter.WithLabelValues(overlays[uIndex].String()).Inc()
		uploadTimeGauge.WithLabelValues(overlays[uIndex].String(), largeFile.Address().String()).Set(d2.Seconds())
		uploadTimeHistogram.Observe(d2.Seconds())
	}

	var (
		counter = 0
		t3      time.Time
	)
DOWNLOAD:
	counter++
	t3 = time.Now()
	smallSize, smallHash, err := c.Nodes[dIndex].DownloadFile(ctx, smallFile.Address())
	if err != nil {
		if counter == 5 {
			metricsHandler(pusher, pushMetrics)
			return fmt.Errorf("node %d: %w", dIndex, err)
		}
		if errors.Is(err, api.ErrServiceUnavailable) {
			time.Sleep(1 * time.Second)
			goto DOWNLOAD
		}
	}
	d3 := time.Since(t3)

	downloadedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
	downloadTimeGauge.WithLabelValues(overlays[dIndex].String(), smallFile.Address().String()).Set(d3.Seconds())
	downloadTimeHistogram.Observe(d3.Seconds())

	if !bytes.Equal(smallFile.Hash(), smallHash) {
		notRetrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
		metricsHandler(pusher, pushMetrics)
		return fmt.Errorf("Node %d. Small file %s (size: %d bytes) not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d", dIndex, smallFile.Address().String(), smallSize, overlays[dIndex].String(), smallFile.Size(), smallSize)
	}

	retrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
	fmt.Printf("Node %d. Small file %s (size: %d bytes) downloaded successfully from node %s\n", dIndex, smallFile.Address().String(), smallSize, overlays[dIndex].String())

	return nil
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
