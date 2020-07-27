package fileretrievaldynamic

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/chaos"
	"github.com/ethersphere/beekeeper/pkg/helm3"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents pushsync check options, there are two conditions:
// 1. 2x DownloadNodeCount + StopNodeCount + 2 <= NodeCount
// 2. 3x DownloadNodeCount + 2x StopNodeCount + 2 <= NodeCount + NewNodeCount
type Options struct {
	DownloadNodeCount int
	FileName          string
	FileSize          int64
	NewNodeCount      int
	Seed              int64
	StopNodeCount     int
	KubeConfig        string
	Namespace         string
	HelmRelease       string
	HelmChart         string
}

var (
	chaosCreate   = "create"
	chaosDelete   = "delete"
	chaosMode     = "one"
	chaosValue    = ""
	chaosPodname  = "bee"
	chaosDuration = "59s"
	chaosCron     = "60s"
)

// var errFileRetrievalDynamic = errors.New("file retrieval dynamic")

// Check uploads file on cluster and downloads them from N random nodes
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(uploadTimeGauge)
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

	// upload file to random node
	uIndex := rnd.Intn(c.Size())
	file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
	t0 := time.Now()
	if err := c.Nodes[uIndex].UploadFile(ctx, &file); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	d0 := time.Since(t0)
	uploadTimeGauge.WithLabelValues(overlays[uIndex].String(), file.Address().String()).Set(d0.Seconds())
	fmt.Printf("Node %d. File %s uploaded successfully to node %s\n", uIndex, file.Address().String(), overlays[uIndex].String())

	// download from random nodes
	d1Skip := []int{uIndex}
	d1Indexes, err := randomIndexes(rnd, o.DownloadNodeCount, c.Size(), d1Skip)
	if err != nil {
		return fmt.Errorf("d1Indexes: %w", err)
	}
	if err := downloadFile(ctx, c, file, d1Indexes, overlays, pusher, pushMetrics); err != nil {
		return err
	}

	// stop random nodes
	s1Skip := []int{0}
	s1Indexes, err := randomIndexes(rnd, o.StopNodeCount, c.Size(), s1Skip)
	if err != nil {
		return fmt.Errorf("s1Indexes: %w", err)
	}
	for _, sIndex := range s1Indexes {
		if err = chaos.PodFailure(ctx, o.KubeConfig, chaosCreate, chaosMode, chaosValue, o.Namespace, fmt.Sprintf("%s-%d", chaosPodname, sIndex), chaosDuration, chaosCron); err != nil {
			return err
		}
		fmt.Printf("Node %s-%d stopped\n", chaosPodname, sIndex)
	}
	fmt.Printf("Waiting for %ds\n", o.StopNodeCount*15)
	time.Sleep(time.Duration(o.StopNodeCount) * 15 * time.Second)

	// download from other random nodes
	d2Skip := []int{uIndex}
	d2Skip = append(d2Skip, d1Indexes...)
	d2Skip = append(d2Skip, s1Indexes...)
	d2Indexes, err := randomIndexes(rnd, o.DownloadNodeCount, c.Size(), d2Skip)
	if err != nil {
		return fmt.Errorf("d2Indexes: %w", err)
	}
	if err := downloadFile(ctx, c, file, d2Indexes, overlays, pusher, pushMetrics); err != nil {
		return err
	}

	// start stopped nodes and download from them
	for _, sIndex := range s1Indexes {
		if err = chaos.PodFailure(ctx, o.KubeConfig, chaosDelete, chaosMode, chaosValue, o.Namespace, fmt.Sprintf("%s-%d", chaosPodname, sIndex), chaosDuration, chaosCron); err != nil {
			return err
		}
		fmt.Printf("Node %s-%d started\n", chaosPodname, sIndex)
	}
	fmt.Printf("Waiting for %ds\n", o.StopNodeCount*30)
	time.Sleep(time.Duration(o.StopNodeCount) * 30 * time.Second)
	if err := downloadFile(ctx, c, file, s1Indexes, overlays, pusher, pushMetrics); err != nil {
		return err
	}

	// add new nodes and download from them
	if o.NewNodeCount > 0 {
		// add nodes to the cluster
		if err := helm3.Upgrade(o.KubeConfig, o.Namespace, o.HelmRelease, o.HelmChart, map[string]string{"set": fmt.Sprintf("replicaCount=%d", c.Size()+o.NewNodeCount)}); err != nil {
			return fmt.Errorf("helm3: %w", err)
		}
		if err := c.AddNodes(o.NewNodeCount); err != nil {
			return fmt.Errorf("adding nodes to the cluster: %w", err)
		}
		fmt.Printf("%d nodes added to the cluster\n", o.NewNodeCount)
		fmt.Printf("Waiting for %ds\n", o.NewNodeCount*60)
		time.Sleep(time.Duration(o.NewNodeCount) * 60 * time.Second)

		overlays, err := c.Overlays(ctx)
		if err != nil {
			return err
		}

		// stop random nodes
		s2Skip := []int{0}
		s2Skip = append(s2Skip, s1Indexes...)
		s2Indexes, err := randomIndexes(rnd, o.StopNodeCount, c.Size(), s2Skip)
		if err != nil {
			return fmt.Errorf("s2Indexes: %w", err)
		}
		for _, sIndex := range s2Indexes {
			if err = chaos.PodFailure(ctx, o.KubeConfig, chaosCreate, chaosMode, chaosValue, o.Namespace, fmt.Sprintf("%s-%d", chaosPodname, sIndex), chaosDuration, chaosCron); err != nil {
				return err
			}
			fmt.Printf("Node %s-%d stopped\n", chaosPodname, sIndex)
		}
		fmt.Printf("Waiting for %ds\n", o.StopNodeCount*15)
		time.Sleep(time.Duration(o.StopNodeCount) * 15 * time.Second)

		// download from other random nodes
		d3Skip := []int{uIndex}
		d3Skip = append(d3Skip, d1Indexes...)
		d3Skip = append(d3Skip, d2Indexes...)
		d3Skip = append(d3Skip, s1Indexes...)
		d3Skip = append(d3Skip, s2Indexes...)
		d3Indexes, err := randomIndexes(rnd, o.DownloadNodeCount, c.Size(), d3Skip)
		if err != nil {
			return fmt.Errorf("d3Indexes: %w", err)
		}
		if err := downloadFile(ctx, c, file, d3Indexes, overlays, pusher, pushMetrics); err != nil {
			return err
		}

		// start stopped nodes and download from them
		for _, sIndex := range s2Indexes {
			if err = chaos.PodFailure(ctx, o.KubeConfig, chaosDelete, chaosMode, chaosValue, o.Namespace, fmt.Sprintf("%s-%d", chaosPodname, sIndex), chaosDuration, chaosCron); err != nil {
				return err
			}
			fmt.Printf("Node %s-%d started\n", chaosPodname, sIndex)
		}
		fmt.Printf("Waiting for %ds\n", o.StopNodeCount*30)
		time.Sleep(time.Duration(o.StopNodeCount) * 30 * time.Second)
		if err := downloadFile(ctx, c, file, s2Indexes, overlays, pusher, pushMetrics); err != nil {
			return err
		}

		// restore cluster to original state
		if err := helm3.Upgrade(o.KubeConfig, o.Namespace, o.HelmRelease, o.HelmChart, map[string]string{"set": fmt.Sprintf("replicaCount=%d", c.Size()-o.NewNodeCount)}); err != nil {
			return fmt.Errorf("helm3: %w", err)
		}
		fmt.Printf("Cluster restored to original state\n")
		if err := c.RemoveNodes(o.NewNodeCount); err != nil {
			return fmt.Errorf("removing nodes from the cluster: %w", err)
		}
	}

	return
}

// downloadFile downloads file from given (indexes) nodes from the cluster
func downloadFile(ctx context.Context, c bee.Cluster, file bee.File, indexes []int, overlays []swarm.Address, pusher *push.Pusher, pushMetrics bool) error {
	for _, i := range indexes {
		t1 := time.Now()
		size, hash, err := c.Nodes[i].DownloadFile(ctx, file.Address())
		if err != nil {
			fmt.Printf("error: node %d: %s\n", i, err)
			// return fmt.Errorf("node %d: %w", i, err)
		}
		d1 := time.Since(t1)

		downloadedCounter.WithLabelValues(overlays[i].String()).Inc()
		downloadTimeGauge.WithLabelValues(overlays[i].String(), file.Address().String()).Set(d1.Seconds())
		downloadTimeHistogram.Observe(d1.Seconds())

		if !bytes.Equal(file.Hash(), hash) {
			notRetrievedCounter.WithLabelValues(overlays[i].String()).Inc()
			fmt.Printf("error: node %d. File %s not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d\n", i, file.Address().String(), overlays[i].String(), file.Size(), size)
			// return errFileRetrievalDynamic
		}
		retrievedCounter.WithLabelValues(overlays[i].String()).Inc()
		fmt.Printf("Node %d. File %s downloaded successfully from node %s\n", i, file.Address().String(), overlays[i].String())

		if pushMetrics {
			if err := pusher.Push(); err != nil {
				fmt.Printf("error: node %d: %s\n", i, err)
			}
		}
	}

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
