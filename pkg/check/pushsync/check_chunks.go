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

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	uploader, err := cluster.RandomNode(ctx, rnd)
	if err != nil {
		return err
	}

	// dont do subsequent checks on uploader
	delete(clients, uploader.Name())

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

	fmt.Printf("uploaded chunk %s to node %s, took %s\n", ref.String(), uploader.Name(), time.Since(start))
	var (
		wg  sync.WaitGroup
		cnt atomic.Int32
	)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-time.After(100 * time.Millisecond):
		}
		if len(clients) == 0 {
			break LOOP
		}
		ctxi, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		for nodeName, client := range clients {
			wg.Add(1)
			c.metrics.RetrieveAttempt.Inc()
			go func(client *bee.Client, name string) {
				defer wg.Done()
				b, err := client.DownloadChunk(ctxi, ref, "")
				if err != nil {
					c.metrics.RetrieveFail.Inc()
					return
				}
				if !bytes.Equal(b, chunk.Data()) {
					c.metrics.RetrieveFail.Inc()
					return
				}

				// mark this node as ok
				idx := cnt.Inc()
				c.metrics.NodeSyncTime.WithLabelValues(fmt.Sprintf("%d", idx)).Add(float64(time.Since(start)))
				fmt.Printf("%d'th node (%s) has chunk, took %s\n", idx, name, time.Since(start))
				delete(clients, name)
			}(client, nodeName)
		}
		fmt.Println("polling")
		wg.Wait()
	}

	if len(clients) > 0 {
		return fmt.Errorf("timed out after %s waiting for chunk to be retrievable, %d nodes remaining", time.Since(start), len(clients))
	}

	return nil
}
