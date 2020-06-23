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
	UploadNodeCount int
	FilesPerNode    int
	FileName        string
	FileSize        int64
	Seed            int64
}

var errFileRetrieval = errors.New("file retrieval")

// Check uploads files on cluster and downloads them from the last node in the cluster
func Check(c bee.Cluster, o Options, pusher *push.Pusher, metrics bool) (err error) {
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

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			t0 := time.Now()
			if err := c.Nodes[i].UploadFile(ctx, &file); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[i].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[i].String(), file.Address().String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(1 * time.Second)
			t1 := time.Now()
			size, hash, err := c.Nodes[c.Size()-1].DownloadFile(ctx, file.Address())
			if err != nil {
				return fmt.Errorf("node %d: %w", c.Size()-1, err)
			}
			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(overlays[i].String()).Inc()
			downloadTimeGauge.WithLabelValues(overlays[i].String(), file.Address().String()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(file.Hash(), hash) {
				notRetrievedCounter.WithLabelValues(overlays[i].String()).Inc()
				fmt.Printf("Node %d. File %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s\n", i, j, file.Size(), size, overlays[i].String(), file.Address().String())
				return errFileRetrieval
			}

			retrievedCounter.WithLabelValues(overlays[i].String()).Inc()
			fmt.Printf("Node %d. File %d retrieved successfully. Node: %s File: %s\n", i, j, overlays[i].String(), file.Address().String())

			if metrics {
				if err := pusher.Push(); err != nil {
					fmt.Printf("node %d: %s\n", i, err)
				}
			}
		}
	}

	return
}

// CheckFull uploads files on cluster and downloads them from the all nodes in the cluster
func CheckFull(c bee.Cluster, o Options, pusher *push.Pusher, metrics bool) (err error) {
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

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		for j := 0; j < o.FilesPerNode; j++ {
			file := bee.NewRandomFile(rnds[i], fmt.Sprintf("%s-%d-%d", o.FileName, i, j), o.FileSize)

			t0 := time.Now()
			if err := c.Nodes[i].UploadFile(ctx, &file); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[i].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[i].String(), file.Address().String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			time.Sleep(1 * time.Second)
			for d := range c.Nodes {
				if d == i {
					continue
				}

				t1 := time.Now()
				size, hash, err := c.Nodes[d].DownloadFile(ctx, file.Address())
				if err != nil {
					return fmt.Errorf("node %d: %w", d, err)
				}
				d1 := time.Since(t1)

				downloadedCounter.WithLabelValues(overlays[i].String()).Inc()
				downloadTimeGauge.WithLabelValues(overlays[i].String(), file.Address().String()).Set(d1.Seconds())
				downloadTimeHistogram.Observe(d1.Seconds())

				if !bytes.Equal(file.Hash(), hash) {
					notRetrievedCounter.WithLabelValues(overlays[i].String()).Inc()
					fmt.Printf("Node %d. File %d not retrieved successfully from node %d. Uploaded size: %d Downloaded size: %d Node: %s Download node: %s File: %s\n", i, j, d, file.Size(), size, overlays[i].String(), overlays[d].String(), file.Address().String())
					return errFileRetrieval
				}

				retrievedCounter.WithLabelValues(overlays[i].String()).Inc()
				fmt.Printf("Node %d. File %d retrieved successfully from node %d. Node: %s Download node: %s File: %s\n", i, j, d, overlays[i].String(), overlays[d].String(), file.Address().String())

				if metrics {
					if err := pusher.Push(); err != nil {
						fmt.Printf("node %d: %s\n", i, err)
					}
				}
			}
		}
	}

	return
}
