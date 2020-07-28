package localpinning

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
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

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	// upload reference file to random node
	uIndex := rnd.Intn(c.Size())
	refFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.SmallFileSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &refFile); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. Reference file %s (size: %d) uploaded successfully to node %s\n", uIndex, refFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	// download reference from random node
	dSkip := []int{uIndex}
	dIndexes, err := randomIndexes(rnd, 1, c.Size(), dSkip)
	dIndex := dIndexes[0]
	refSize, refHash, err := c.Nodes[dIndex].DownloadFile(ctx, refFile.Address())
	if err != nil {
		return fmt.Errorf("node %d: %w", dIndex, err)
	}
	if !bytes.Equal(refFile.Hash(), refHash) {
		return fmt.Errorf("Node %d. Reference file %s (size: %d) not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d", dIndex, refFile.Address().String(), refSize, overlays[dIndex].String(), refFile.Size(), refSize)
	}
	fmt.Printf("Node %d. Reference file %s (size: %d) downloaded successfully from node %s\n", dIndex, refFile.Address().String(), refSize, overlays[dIndex].String())

	smallFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.SmallFileSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &smallFile); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. Small file %s (size: %d) uploaded successfully to node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	pinned, err := c.Nodes[uIndex].PinChunk(ctx, smallFile.Address())
	if err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	if !pinned {
		// notSyncedCounter.WithLabelValues(overlays[i].String()).Inc()
		return fmt.Errorf("Node %d. Small file %s (size: %d) not pinned successfully on node %s", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())
	}
	fmt.Printf("Node %d. Small file %s (size: %d) pinned successfully on node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

	for i := 0; i < o.LargeFileCount; i++ {
		largeFile := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.LargeFileSize)
		if err := c.Nodes[uIndex].UploadFile(ctx, &largeFile); err != nil {
			return fmt.Errorf("node %d: %w", uIndex, err)
		}
		fmt.Printf("Node %d. Large file %s (size: %d) uploaded successfully to node %s\n", uIndex, largeFile.Address().String(), o.LargeFileSize, overlays[uIndex].String())

		smallSize, smallHash, err := c.Nodes[dIndex].DownloadFile(ctx, smallFile.Address())
		if err != nil {
			return fmt.Errorf("node %d: %w", dIndex, err)
		}
		if !bytes.Equal(smallFile.Hash(), smallHash) {
			return fmt.Errorf("Node %d. Small file %s (size: %d) not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d", dIndex, smallFile.Address().String(), smallSize, overlays[dIndex].String(), smallFile.Size(), smallSize)
		}
		fmt.Printf("Node %d. Small file %s (size: %d) downloaded successfully from node %s\n", dIndex, smallFile.Address().String(), smallSize, overlays[dIndex].String())
	}

	pins, err := c.Nodes[uIndex].PinnedChunks(ctx)
	if err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	pinFound := false
	for _, pin := range pins.Chunks {
		if pin.Address.Equal(smallFile.Address()) {
			pinFound = true
		}
	}
	if !pinFound {
		return fmt.Errorf("Node %d. Small file %s not found in the pinned chunk list", uIndex, smallFile.Address().String())
	}

	// cleanup
	unpinned, err := c.Nodes[uIndex].UnpinChunk(ctx, smallFile.Address())
	if err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	if !unpinned {
		// notSyncedCounter.WithLabelValues(overlays[i].String()).Inc()
		return fmt.Errorf("Node %d. Small file %s (size: %d) not unpinned successfully on node %s", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())
	}
	fmt.Printf("Node %d. Small file %s (size: %d) unpinned successfully on node %s\n", uIndex, smallFile.Address().String(), o.SmallFileSize, overlays[uIndex].String())

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
