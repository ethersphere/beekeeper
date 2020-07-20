package fileretrievaldynamic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/chaos"
	"github.com/ethersphere/beekeeper/pkg/helm3"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents pushsync check options
type Options struct {
	DownloadNodeCount int
	FileName          string
	FileSize          int64
	NewNodeCount      int
	Seed              int64
	StopNodeCount     int
}

// ChaosOptions ...
type ChaosOptions struct {
	Kubeconfig string
	Action     string
	Mode       string
	Value      string
	Namespace  string
	Podname    string
	Duration   string
	Cron       string
}

// HelmOptions ...
type HelmOptions struct {
	Kubeconfig string
	Namespace  string
	Release    string
	Chart      string
	Args       map[string]string
}

var chaosStart = ChaosOptions{
	Kubeconfig: "/Users/svetomir.smiljkovic/.kube/config",
	Action:     "create",
	Mode:       "one",
	Value:      "",
	Namespace:  "svetomir",
	Podname:    "bee",
	Duration:   "59s",
	Cron:       "60s",
}

var chaosStop = ChaosOptions{
	Kubeconfig: "/Users/svetomir.smiljkovic/.kube/config",
	Action:     "delete",
	Mode:       "one",
	Value:      "",
	Namespace:  "svetomir",
	Podname:    "bee",
	Duration:   "59s",
	Cron:       "60s",
}

var helmScale = HelmOptions{
	Kubeconfig: "/Users/svetomir.smiljkovic/.kube/config",
	Namespace:  "svetomir",
	Release:    "bee",
	Chart:      "ethersphere/bee",
	Args:       map[string]string{"set": "replicaCount=20"},
}

var helmRestore = HelmOptions{
	Kubeconfig: "/Users/svetomir.smiljkovic/.kube/config",
	Namespace:  "svetomir",
	Release:    "bee",
	Chart:      "ethersphere/bee",
	Args:       map[string]string{"set": "replicaCount=15"},
}

var errFileRetrievalDynamic = errors.New("file retrieval dynamic")

// Check uploads file on cluster and downloads them from N random nodes
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	uIndex := rnd.Intn(c.Size())
	file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &file); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. File %s uploaded successfully to node %s\n", uIndex, file.Address().String(), overlays[uIndex].String())

	// download from unaltered cluster
	dIndexes := []int{}
	found := false
	for !found {
		i := rnd.Intn(c.Size())
		if uIndex != i && !contains(dIndexes, i) {
			dIndexes = append(dIndexes, i)
		}
		if len(dIndexes) == o.DownloadNodeCount {
			found = true
		}
	}
	for _, dIndex := range dIndexes {
		size, hash, err := c.Nodes[dIndex].DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %d: %w", dIndex, err)
		}

		if !bytes.Equal(file.Hash(), hash) {
			notRetrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
			fmt.Printf("Node %d. File %s not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d\n", dIndex, file.Address().String(), overlays[dIndex].String(), file.Size(), size)
			return errFileRetrievalDynamic
		}
		fmt.Printf("Node %d. File %s downloaded successfully from node %s\n", dIndex, file.Address().String(), overlays[dIndex].String())
	}

	// stop nodes
	sIndexes := []int{}
	found = false
	for !found {
		i := rnd.Intn(c.Size())
		if i != 0 && !contains(sIndexes, i) {
			sIndexes = append(sIndexes, i)
		}
		if len(sIndexes) == o.StopNodeCount {
			found = true
		}
	}
	fmt.Printf("Alter cluster\n")
	for _, sIndex := range sIndexes {
		if err = chaos.PodFailure(ctx, chaosStart.Kubeconfig, chaosStart.Action, chaosStart.Mode, chaosStart.Value, chaosStart.Namespace, fmt.Sprintf("%s-%d", chaosStart.Podname, sIndex), chaosStart.Duration, chaosStart.Cron); err != nil {
			return err
		}
		fmt.Printf("Node %s-%d stopped\n", chaosStart.Podname, sIndex)
	}
	time.Sleep(60 * time.Second)

	// download from cluster with stopped nodes
	d2Indexes := []int{}
	found = false
	for !found {
		i := rnd.Intn(c.Size())
		if uIndex != i && !contains(dIndexes, i) && !contains(d2Indexes, i) && !contains(sIndexes, i) {
			d2Indexes = append(d2Indexes, i)
		}
		if len(d2Indexes) == o.DownloadNodeCount {
			found = true
		}
	}
	for _, dIndex := range d2Indexes {
		if contains(sIndexes, dIndex) {
			fmt.Printf("Node %d. Stopped. Node %s\n", dIndex, overlays[dIndex].String())
			continue
		}
		size, hash, err := c.Nodes[dIndex].DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %d: %w", dIndex, err)
		}

		if !bytes.Equal(file.Hash(), hash) {
			notRetrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
			fmt.Printf("Node %d. File %s not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d\n", dIndex, file.Address().String(), overlays[dIndex].String(), file.Size(), size)
			return errFileRetrievalDynamic
		}
		fmt.Printf("Node %d. File %s downloaded successfully from node %s\n", dIndex, file.Address().String(), overlays[dIndex].String())
	}

	// add new nodes and download
	if o.NewNodeCount > 0 {
		if err := helm3.Upgrade(helmScale.Kubeconfig, helmScale.Namespace, helmScale.Release, helmScale.Chart, helmScale.Args); err != nil {
			return fmt.Errorf("helm3: %w", err)
		}
		fmt.Println("cluster scaled")
		// TODO: download from all new nodes
		// time.Sleep(120 * time.Second)
	}

	// start stopped nodes and download
	fmt.Printf("Restore cluster\n")
	for _, sIndex := range sIndexes {
		if err = chaos.PodFailure(ctx, chaosStop.Kubeconfig, chaosStop.Action, chaosStop.Mode, chaosStop.Value, chaosStop.Namespace, fmt.Sprintf("%s-%d", chaosStop.Podname, sIndex), chaosStop.Duration, chaosStop.Cron); err != nil {
			return err
		}
		fmt.Printf("Node %s-%d started\n", chaosStop.Podname, sIndex)
	}
	time.Sleep(120 * time.Second)

	d4Indexes := append(dIndexes, d2Indexes...)
	for _, dIndex := range d4Indexes {
		size, hash, err := c.Nodes[dIndex].DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %d: %w", dIndex, err)
		}

		if !bytes.Equal(file.Hash(), hash) {
			notRetrievedCounter.WithLabelValues(overlays[dIndex].String()).Inc()
			fmt.Printf("Node %d. File %s not downloaded successfully from node %s. Uploaded size: %d Downloaded size: %d\n", dIndex, file.Address().String(), overlays[dIndex].String(), file.Size(), size)
			return errFileRetrievalDynamic
		}
		fmt.Printf("Node %d. File %s downloaded successfully from node %s\n", dIndex, file.Address().String(), overlays[dIndex].String())
	}

	// remove new nodes and download
	if o.NewNodeCount > 0 {
		if err := helm3.Upgrade(helmRestore.Kubeconfig, helmRestore.Namespace, helmRestore.Release, helmRestore.Chart, helmRestore.Args); err != nil {
			return fmt.Errorf("helm3: %w", err)
		}
		fmt.Println("cluster restored")
		// TODO: download from all inital nodes
		// time.Sleep(120 * time.Second)
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
