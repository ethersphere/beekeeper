package pushsync

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"go.uber.org/atomic"
)

// checkChunks uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) checkChunks(ctx context.Context, cluster *bee.Cluster, o Options) error {
	fmt.Println("running pushsync (chunks mode)")
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("seed: %d\n", o.Seed)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	clusterNodes := cluster.Nodes()
	for _, uploader := range clusterNodes {
		batchID, err := uploader.Client().GetOrCreateBatch(ctx, 100000000, 20, "", "")
		if err != nil {
			return fmt.Errorf("uploader node %s: get or create batch: %w", uploader.Name(), err)
		}

		chunk, err := bee.NewRandomChunk(rnd)
		if err != nil {
			return fmt.Errorf("node %s: %w", uploader.Name(), err)
		}

		start := time.Now()

		ref, err := uploader.Client().UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
		if err != nil {
			return fmt.Errorf("node %s: %w", uploader.Name(), err)
		}

		c.metrics.UploadSuccess.WithLabelValues(uploader.Name()).Inc()

		fmt.Printf("uploaded chunk %s to node %s, took %s\n", ref.String(), uploader.Name(), time.Since(start))

		var (
			wg  sync.WaitGroup
			cnt atomic.Int32
		)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP
			case <-time.After(100 * time.Millisecond):
			}

			targetNodes := filterOut(clusterNodes, uploader)

			if int(cnt.Load()) == len(targetNodes) {
				break LOOP
			}

			for nodeName, node := range targetNodes {
				wg.Add(1)
				c.metrics.RetrieveAttempt.WithLabelValues(nodeName).Inc()
				go func(client *bee.Client, name string) {
					defer wg.Done()

					ctxi, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()

					b, err := client.DownloadChunk(ctxi, ref, "")
					if err != nil {
						c.metrics.RetrieveFail.WithLabelValues(name).Inc()
						return
					}
					if !bytes.Equal(b, chunk.Data()) {
						c.metrics.RetrieveFail.WithLabelValues(name).Inc()
						return
					}

					idx := cnt.Inc()
					c.metrics.NodeSyncTime.WithLabelValues(fmt.Sprintf("%d", idx)).Add(float64(time.Since(start)))
					fmt.Printf("%d'th node (%s) has chunk, took %s\n", idx, name, time.Since(start))
				}(node.Client(), nodeName)
			}
			fmt.Println("polling")
			wg.Wait()

			if int(cnt.Load()) != len(targetNodes) {
				return fmt.Errorf("timed out after %s waiting for chunk to be retrievable, %d nodes remaining", time.Since(start), len(targetNodes)-int(cnt.Load()))
			}

			c.metrics.RetrieveSuccess.WithLabelValues(uploader.Name()).Inc()
		}
	}

	return nil
}

func filterOut(all map[string]*bee.Node, exclude *bee.Node) (out map[string]*bee.Node) {
	out = make(map[string]*bee.Node)
	for k, v := range all {
		if v == exclude {
			continue
		}
		out[k] = v
	}
	return
}
