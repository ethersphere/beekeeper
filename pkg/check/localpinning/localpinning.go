package localpinning

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents localpinning check options
type Options struct {
	DiskSize int64
	FileName string
	Seed     int64
}

var errLocalPinning = errors.New("local pinning")

// Check ...
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	// upload file to random node
	uIndex := rnd.Intn(c.Size())
	file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.DiskSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &file); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. File %s uploaded successfully to node %s\n", uIndex, file.Address().String(), overlays[uIndex].String())

	// download from random node
	dSkip := []int{uIndex}
	dIndexes, err := randomIndexes(rnd, 1, c.Size(), dSkip)
	dIndex := dIndexes[0]
	size, hash, err := c.Nodes[dIndex].DownloadFile(ctx, file.Address())
	if err != nil {
		return fmt.Errorf("node %d: %w", dIndex, err)
	}
	if !bytes.Equal(file.Hash(), hash) {
		fmt.Printf("error: node %d. File %s not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d\n", dIndex, file.Address().String(), overlays[dIndex].String(), file.Size(), size)
		return errLocalPinning
	}
	fmt.Printf("Node %d. File %s downloaded successfully from node %s\n", dIndex, file.Address().String(), overlays[dIndex].String())

	// Pinning Testcase 1
	file11 := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.DiskSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &file11); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. File %s uploaded successfully to node %s\n", uIndex, file11.Address().String(), overlays[uIndex].String())

	pinned, err := c.Nodes[uIndex].PinChunk(ctx, file11.Address())
	if err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	if !pinned {
		// notSyncedCounter.WithLabelValues(overlays[i].String()).Inc()
		fmt.Printf("Node %d. File %s not pinned\n", uIndex, file11.Address().String())
		return errLocalPinning
	}

	file12 := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.DiskSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &file12); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. File %s uploaded successfully to node %s\n", uIndex, file12.Address().String(), overlays[uIndex].String())

	size11, hash11, err := c.Nodes[dIndex].DownloadFile(ctx, file11.Address())
	if err != nil {
		return fmt.Errorf("node %d: %w", dIndex, err)
	}
	if !bytes.Equal(file11.Hash(), hash11) {
		fmt.Printf("error: node %d. File %s not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d\n", dIndex, file11.Address().String(), overlays[dIndex].String(), file11.Size(), size11)
		return errLocalPinning
	}
	fmt.Printf("Node %d. File %s downloaded successfully from node %s\n", dIndex, file11.Address().String(), overlays[dIndex].String())

	pins, err := c.Nodes[uIndex].PinnedChunks(ctx)
	if err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	pinFound := false
	for _, pin := range pins.Chunks {
		if pin.Address.Equal(file11.Address()) {
			pinFound = true
		}
	}
	if !pinFound {
		fmt.Printf("Node %d. File %s not found in the pinned chunk list\n", uIndex, file11.Address().String())
		return errLocalPinning
	}

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
