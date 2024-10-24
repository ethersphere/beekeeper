package networkavailability

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/storage/testing"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents smoke test options
type Options struct {
	RndSeed       int64
	PostageAmount int64
	PostageDepth  uint64
	SleepDuration time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		RndSeed:       time.Now().UnixNano(),
		PostageAmount: 50_000_000,
		PostageDepth:  24,
		SleepDuration: time.Hour,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics(),
		logger:  logger,
	}
}

// Run creates file of specified size that is uploaded and downloaded.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	c.logger.Info("random seed: ", o.RndSeed)
	rnd := random.PseudoGenerator(o.RndSeed)

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

iteration:
	for i := 0; true; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("starting iteration: #%d", i)
		}

		perm := rnd.Perm(cluster.Size())
		names := cluster.NodeNames()
		uploadNode := names[perm[0]]
		downloadNode := names[perm[1]]

		// if the upload and download nodes are the same, try again for a different peer
		if uploadNode == downloadNode {
			continue
		}

		uploadClient := clients[uploadNode]
		downloadClient := clients[uploadNode]
		state, err := uploadClient.ReserveState(ctx)
		if err != nil {
			c.logger.Error("reserve state failure", err)
		}
		storageRadius := state.StorageRadius

		c.logger.Infof("uploder node: %s", uploadNode)
		c.logger.Infof("downloader node: %s", downloadNode)
		c.logger.Infof("storage radius: %d", storageRadius)

		// upload
		var chunks []swarm.Chunk
		for _, n := range neighborhoods(int(storageRadius)) {

			batch, err := uploadClient.GetOrCreateMutableBatch(ctx, o.PostageAmount, o.PostageDepth, "net-avail-check")
			if err != nil {
				c.logger.Errorf("create batch failed failed")
				continue iteration
			}

			// mine chunk
			ch := testing.GenerateValidRandomChunkAt(n, int(storageRadius))
			c.metrics.UploadAttempts.Inc()
			t := time.Now()

			// upload chunk
			resp, err := uploadClient.UploadChunk(ctx, ch.Data(), api.UploadOptions{BatchID: batch, Direct: true})
			if err != nil {
				c.logger.Errorf("upload failed to neighborhood %s: %v", n, err)
				c.metrics.UploadErrors.Inc()
				c.metrics.UploadDuration.WithLabelValues("false").Observe(time.Since(t).Seconds())
			} else if !resp.Equal(ch.Address()) {
				c.logger.Errorf("uploaded chunk and response addresses do no match, uploaded %s, downloaded %s", ch, resp)
			} else {
				c.metrics.UploadDuration.WithLabelValues("true").Observe(time.Since(t).Seconds())
				chunks = append(chunks, ch)
			}
		}

		c.logger.Infof("uploaded to %d neighborhoods, starting downloading", len(chunks))

		for _, ch := range chunks {

			t := time.Now()

			c.metrics.DownloadAttempts.Inc()

			data, err := downloadClient.DownloadChunk(ctx, ch.Address(), "", nil)
			if err != nil {
				c.metrics.DownloadErrors.Inc()
				c.metrics.DownloadDuration.WithLabelValues("false").Observe(time.Since(t).Seconds())
				c.logger.Errorf("download failed: %v", err)
			} else if !bytes.Equal(data, ch.Data()) {
				c.logger.Errorf("uploaded chunk and response data do no match for chunk_address %s", ch.Address())
			} else {
				c.metrics.DownloadDuration.WithLabelValues("true").Observe(time.Since(t).Seconds())
			}
		}

		c.logger.Info("download finished")

		time.Sleep(o.SleepDuration)
	}

	return nil
}

func neighborhoods(bits int) []swarm.Address {
	max := 1 << bits
	leftover := bits % 8

	ret := make([]swarm.Address, 0, max)

	for i := 0; i < max; i++ {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(i))

		var addr []byte

		if bits <= 8 {
			addr = []byte{buf[0]}
		} else if bits <= 16 {
			addr = []byte{buf[0], buf[1]}
		} else if bits <= 24 {
			addr = []byte{buf[0], buf[1], buf[2]}
		} else if bits <= 32 {
			addr = []byte{buf[0], buf[1], buf[2], buf[3]}
		}

		if leftover > 0 {
			addr[len(addr)-1] <<= (8 - leftover)
		}

		ret = append(ret, bytesToAddr(addr))
	}

	return ret
}

func bytesToAddr(b []byte) swarm.Address {
	addr := make([]byte, swarm.HashSize)
	copy(addr, b)
	return swarm.NewAddress(addr)
}
