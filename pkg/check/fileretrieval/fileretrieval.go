package fileretrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents pushsync check options
type Options struct {
	NodeGroup       string
	UploadNodeCount int
	FilesPerNode    int
	FileName        string
	FileSize        int64
	Seed            int64
}

var errFileRetrieval = errors.New("file retrieval")

// Check uploads files on cluster and downloads them from the last node in the cluster
func Check(c *bee.DynamicCluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(downloadedCounter)
	pusher.Collector(downloadTimeGauge)
	pusher.Collector(downloadTimeHistogram)
	pusher.Collector(retrievedCounter)
	pusher.Collector(notRetrievedCounter)

	pusher.Format(expfmt.FmtText)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	lastNodeName := sortedNodes[len(sortedNodes)-1]
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			t0 := time.Now()
			if err := ng.Node(nodeName).UploadFile(ctx, &file, false); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(1 * time.Second)
			t1 := time.Now()
			size, hash, err := ng.Node(lastNodeName).DownloadFile(ctx, file.Address())
			if err != nil {
				return fmt.Errorf("node %s: %w", lastNodeName, err)
			}
			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			downloadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(file.Hash(), hash) {
				notRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. File %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s\n", nodeName, j, file.Size(), size, overlays[nodeName].String(), file.Address().String())
				return errFileRetrieval
			}

			retrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			fmt.Printf("Node %s. File %d retrieved successfully. Node: %s File: %s\n", nodeName, j, overlays[nodeName].String(), file.Address().String())

			if pushMetrics {
				if err := pusher.Push(); err != nil {
					fmt.Printf("node %s: %s\n", nodeName, err)
				}
			}
		}
	}

	return
}

// CheckFull uploads files on cluster and downloads them from the all nodes in the cluster
func CheckFull(c *bee.DynamicCluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadedCounter)
	pusher.Collector(uploadTimeGauge)
	pusher.Collector(uploadTimeHistogram)
	pusher.Collector(downloadedCounter)
	pusher.Collector(downloadTimeGauge)
	pusher.Collector(downloadTimeHistogram)
	pusher.Collector(retrievedCounter)
	pusher.Collector(notRetrievedCounter)

	pusher.Format(expfmt.FmtText)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			t0 := time.Now()
			if err := ng.Node(nodeName).UploadFile(ctx, &file, false); err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(1 * time.Second)
			for n, c := range ng.Nodes() {
				if n == nodeName {
					continue
				}

				t1 := time.Now()
				size, hash, err := c.DownloadFile(ctx, file.Address())
				if err != nil {
					return fmt.Errorf("node %s: %w", n, err)
				}
				d1 := time.Since(t1)

				downloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				downloadTimeGauge.WithLabelValues(overlays[nodeName].String(), file.Address().String()).Set(d1.Seconds())
				downloadTimeHistogram.Observe(d1.Seconds())

				if !bytes.Equal(file.Hash(), hash) {
					notRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
					fmt.Printf("Node %s. File %d not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d Node: %s Download node: %s File: %s\n", nodeName, j, n, file.Size(), size, overlays[nodeName].String(), overlays[n].String(), file.Address().String())
					return errFileRetrieval
				}

				retrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. File %d retrieved successfully from node %s. Node: %s Download node: %s File: %s\n", nodeName, j, n, overlays[nodeName].String(), overlays[n].String(), file.Address().String())

				if pushMetrics {
					if err := pusher.Push(); err != nil {
						fmt.Printf("node %s: %s\n", nodeName, err)
					}
				}
			}
		}
	}

	return
}
