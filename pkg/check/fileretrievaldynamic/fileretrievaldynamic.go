package fileretrievaldynamic

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/chaos"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents pushsync check options
type Options struct {
	DownloadNodeCount int
	FilesPerNode      int
	FileName          string
	FileSize          int64
	Seed              int64
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

	file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
	if err := c.Nodes[uIndex].UploadFile(ctx, &file); err != nil {
		return fmt.Errorf("node %d: %w", uIndex, err)
	}
	fmt.Printf("Node %d. File %s uploaded successfully to node %s\n", uIndex, file.Address().String(), overlays[uIndex].String())

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

	fmt.Printf("Alter cluster\n")
	fmt.Printf("Bee-1 fail\n")

	kubeconfig := "~/.kube/config"
	action := "create"
	mode := "one"
	value := ""
	namespace := "svetomir"
	podname := "bee-1"
	duration := "59.99s"
	cron := "60s"

	err = chaos.PodFailure(ctx, kubeconfig, action, mode, value, namespace, podname, duration, cron)
	if err != nil {
		return err
	}

	fmt.Printf("Bee-1 back\n")

	kubeconfig = "~/.kube/config"
	action = "delete"
	mode = "one"
	value = ""
	namespace = "svetomir"
	podname = "bee-1"
	duration = "59.99s"
	cron = "60s"

	err = chaos.PodFailure(ctx, kubeconfig, action, mode, value, namespace, podname, duration, cron)
	if err != nil {
		return err
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
