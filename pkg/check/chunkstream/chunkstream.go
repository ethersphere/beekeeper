package chunkstream

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	pb "github.com/ethersphere/beekeeper/pkg/bee/api/chunkstreampb"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	ChunkCount     int // CACs uploaded over the stream, in addition to one SOC
	PostageTTL     time.Duration
	PostageDepth   uint64
	PostageLabel   string
	RequestTimeout time.Duration // per websocket read or write
	RetryCount     int           // download rounds for chunks reported as not found or unavailable
	RetryDelay     time.Duration
	Seed           int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunkCount:     10,
		PostageTTL:     24 * time.Hour,
		PostageDepth:   17,
		PostageLabel:   "test-label",
		RequestTimeout: 5 * time.Minute,
		RetryCount:     3,
		RetryDelay:     5 * time.Second,
		Seed:           random.Int64(),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	logger logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

var errChunkStream = errors.New("chunk stream")

type chunk struct {
	addr swarm.Address
	data []byte
}

// Run uploads chunks to one full node over /chunks/stream and downloads them
// from another full node over the same endpoint. It also checks a per-request
// batch, a bad request that must not end the stream, and a chunk that does not
// exist.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if o.ChunkCount < 1 {
		return fmt.Errorf("chunk-count must be at least 1, got %d", o.ChunkCount)
	}
	if o.RetryCount < 1 {
		return fmt.Errorf("retry-count must be at least 1, got %d", o.RetryCount)
	}

	rnd := random.PseudoGenerator(o.Seed)
	c.logger.Infof("seed: %d", o.Seed)

	clients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return err
	}
	if len(clients) < 2 {
		return fmt.Errorf("chunk stream check requires at least 2 full nodes, got %d", len(clients))
	}
	uploader, downloader := clients[0], clients[1]
	c.logger.Infof("uploading to node %s, downloading from node %s", uploader.Name(), downloader.Name())

	batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: create batch: %w", uploader.Name(), err)
	}
	c.logger.Infof("node %s: using batch %s", uploader.Name(), batchID)

	chunks := make([]chunk, 0, o.ChunkCount+1)
	for range o.ChunkCount {
		ch, err := randomCAC(rnd)
		if err != nil {
			return err
		}
		chunks = append(chunks, ch)
	}
	sch, err := randomSOC(rnd)
	if err != nil {
		return err
	}
	chunks = append(chunks, sch)

	if err := c.upload(ctx, uploader, batchID, chunks, o); err != nil {
		return fmt.Errorf("node %s: upload: %w", uploader.Name(), err)
	}

	extra, err := randomCAC(rnd)
	if err != nil {
		return err
	}
	if err := c.uploadPerRequestBatch(ctx, uploader, batchID, extra, rnd, o); err != nil {
		return fmt.Errorf("node %s: per-request batch: %w", uploader.Name(), err)
	}
	chunks = append(chunks, extra)

	if err := c.download(ctx, downloader, chunks, o); err != nil {
		return fmt.Errorf("node %s: download: %w", downloader.Name(), err)
	}

	if err := c.downloadMissing(ctx, downloader, rnd, o); err != nil {
		return fmt.Errorf("node %s: missing chunk: %w", downloader.Name(), err)
	}

	return nil
}

// upload pipelines a Put for every chunk on a stream pinned to batchID and
// expects STATUS_OK for each.
func (c *Check) upload(ctx context.Context, node *bee.Client, batchID string, chunks []chunk, o Options) error {
	stream, err := node.API().ChunkStream.Open(ctx, batchID)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer c.closeStream(node, stream)

	pending := make(map[string]struct{}, len(chunks))
	for _, ch := range chunks {
		if err := stream.Put(ch.addr, ch.data, nil, o.RequestTimeout); err != nil {
			return fmt.Errorf("chunk %s: %w", ch.addr, err)
		}
		pending[ch.addr.ByteString()] = struct{}{}
	}

	start := time.Now()
	for len(pending) > 0 {
		r, err := c.receivePut(stream, o)
		if err != nil {
			return fmt.Errorf("%d of %d replies outstanding: %w", len(pending), len(chunks), err)
		}
		addr := swarm.NewAddress(r.Address)
		if _, ok := pending[addr.ByteString()]; !ok {
			return fmt.Errorf("%w: unexpected or duplicate reply for %s", errChunkStream, addr)
		}
		delete(pending, addr.ByteString())
		if r.Status != pb.Status_STATUS_OK {
			return fmt.Errorf("%w: chunk %s: %s: %s", errChunkStream, addr, r.Status, r.Error)
		}
	}
	c.logger.Infof("node %s: uploaded %d chunks in %v", node.Name(), len(chunks), time.Since(start))

	return nil
}

// uploadPerRequestBatch opens a stream without a connection batch. A Put whose
// address does not match its data must get STATUS_BAD_REQUEST without ending
// the stream, and a Put that names batchID must then get STATUS_OK on the same
// stream.
func (c *Check) uploadPerRequestBatch(ctx context.Context, node *bee.Client, batchID string, ch chunk, rnd *rand.Rand, o Options) error {
	batch, err := hex.DecodeString(batchID)
	if err != nil {
		return fmt.Errorf("decode batch id %s: %w", batchID, err)
	}

	stream, err := node.API().ChunkStream.Open(ctx, "")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer c.closeStream(node, stream)

	wrongAddr := make([]byte, swarm.HashSize)
	if _, err := rnd.Read(wrongAddr); err != nil {
		return fmt.Errorf("create address: %w", err)
	}
	if err := stream.Put(swarm.NewAddress(wrongAddr), ch.data, batch, o.RequestTimeout); err != nil {
		return fmt.Errorf("put with wrong address: %w", err)
	}
	r, err := c.receivePut(stream, o)
	if err != nil {
		return fmt.Errorf("put with wrong address: %w", err)
	}
	if r.Status != pb.Status_STATUS_BAD_REQUEST {
		return fmt.Errorf("%w: put with wrong address: expected %s, got %s: %s", errChunkStream, pb.Status_STATUS_BAD_REQUEST, r.Status, r.Error)
	}

	if err := stream.Put(ch.addr, ch.data, batch, o.RequestTimeout); err != nil {
		return fmt.Errorf("chunk %s: %w", ch.addr, err)
	}
	r, err = c.receivePut(stream, o)
	if err != nil {
		return fmt.Errorf("chunk %s: %w", ch.addr, err)
	}
	if !bytes.Equal(r.Address, ch.addr.Bytes()) {
		return fmt.Errorf("%w: expected a reply for %s, got one for %x", errChunkStream, ch.addr, r.Address)
	}
	if r.Status != pb.Status_STATUS_OK {
		return fmt.Errorf("%w: chunk %s: %s: %s", errChunkStream, ch.addr, r.Status, r.Error)
	}
	c.logger.Infof("node %s: bad request kept the stream open; per-request batch upload ok", node.Name())

	return nil
}

// download pipelines a Get for every chunk and expects each one back with the
// uploaded bytes. Chunks reported as not found or unavailable are requested
// again, up to RetryCount rounds in total.
func (c *Check) download(ctx context.Context, node *bee.Client, chunks []chunk, o Options) error {
	stream, err := node.API().ChunkStream.Open(ctx, "")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer c.closeStream(node, stream)

	want := make(map[string][]byte, len(chunks))
	for _, ch := range chunks {
		want[ch.addr.ByteString()] = ch.data
	}

	start := time.Now()
	todo := chunks
	for round := 1; ; round++ {
		pending := make(map[string]struct{}, len(todo))
		for _, ch := range todo {
			if err := stream.Get(ch.addr, pb.CacheOption_CACHE_DISABLE, o.RequestTimeout); err != nil {
				return fmt.Errorf("chunk %s: %w", ch.addr, err)
			}
			pending[ch.addr.ByteString()] = struct{}{}
		}

		var retry []chunk
		var lastErr string
		for len(pending) > 0 {
			r, err := c.receiveGet(stream, o)
			if err != nil {
				return fmt.Errorf("%d of %d replies outstanding: %w", len(pending), len(todo), err)
			}
			addr := swarm.NewAddress(r.Address)
			if _, ok := pending[addr.ByteString()]; !ok {
				return fmt.Errorf("%w: unexpected or duplicate reply for %s", errChunkStream, addr)
			}
			delete(pending, addr.ByteString())

			switch r.Status {
			case pb.Status_STATUS_OK:
				if !bytes.Equal(r.Data, want[addr.ByteString()]) {
					return fmt.Errorf("%w: chunk %s: downloaded data differs from uploaded data", errChunkStream, addr)
				}
			case pb.Status_STATUS_NOT_FOUND, pb.Status_STATUS_UNAVAILABLE:
				retry = append(retry, chunk{addr: addr, data: want[addr.ByteString()]})
				lastErr = fmt.Sprintf("%s: %s", r.Status, r.Error)
			default:
				return fmt.Errorf("%w: chunk %s: %s: %s", errChunkStream, addr, r.Status, r.Error)
			}
		}

		if len(retry) == 0 {
			break
		}
		if round == o.RetryCount {
			return fmt.Errorf("%w: %d of %d chunks not retrieved after %d rounds, last: %s", errChunkStream, len(retry), len(chunks), round, lastErr)
		}
		c.logger.Infof("node %s: %d chunks not retrieved in round %d, retrying in %v", node.Name(), len(retry), round, o.RetryDelay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(o.RetryDelay):
		}
		todo = retry
	}
	c.logger.Infof("node %s: downloaded %d chunks in %v", node.Name(), len(chunks), time.Since(start))

	return nil
}

// downloadMissing requests an address that was never uploaded and expects
// STATUS_NOT_FOUND or STATUS_UNAVAILABLE, never data or an error.
func (c *Check) downloadMissing(ctx context.Context, node *bee.Client, rnd *rand.Rand, o Options) error {
	missing, err := randomCAC(rnd)
	if err != nil {
		return err
	}

	stream, err := node.API().ChunkStream.Open(ctx, "")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer c.closeStream(node, stream)

	if err := stream.Get(missing.addr, pb.CacheOption_CACHE_DEFAULT, o.RequestTimeout); err != nil {
		return fmt.Errorf("chunk %s: %w", missing.addr, err)
	}
	r, err := c.receiveGet(stream, o)
	if err != nil {
		return fmt.Errorf("chunk %s: %w", missing.addr, err)
	}
	if !bytes.Equal(r.Address, missing.addr.Bytes()) {
		return fmt.Errorf("%w: expected a reply for %s, got one for %x", errChunkStream, missing.addr, r.Address)
	}
	if r.Status != pb.Status_STATUS_NOT_FOUND && r.Status != pb.Status_STATUS_UNAVAILABLE {
		return fmt.Errorf("%w: chunk %s: expected %s or %s, got %s: %s", errChunkStream, missing.addr,
			pb.Status_STATUS_NOT_FOUND, pb.Status_STATUS_UNAVAILABLE, r.Status, r.Error)
	}
	c.logger.Infof("node %s: missing chunk reported as %s", node.Name(), r.Status)

	return nil
}

func (c *Check) receivePut(stream *api.ChunkStream, o Options) (*pb.PutResponse, error) {
	resp, err := stream.Receive(o.RequestTimeout)
	if err != nil {
		return nil, err
	}
	r := resp.GetPut()
	if r == nil {
		return nil, fmt.Errorf("%w: expected a put reply, got a get reply", errChunkStream)
	}
	return r, nil
}

func (c *Check) receiveGet(stream *api.ChunkStream, o Options) (*pb.GetResponse, error) {
	resp, err := stream.Receive(o.RequestTimeout)
	if err != nil {
		return nil, err
	}
	r := resp.GetGet()
	if r == nil {
		return nil, fmt.Errorf("%w: expected a get reply, got a put reply", errChunkStream)
	}
	return r, nil
}

func (c *Check) closeStream(node *bee.Client, stream *api.ChunkStream) {
	if err := stream.Close(); err != nil {
		c.logger.Debugf("node %s: close stream: %v", node.Name(), err)
	}
}

// randomCAC returns a content-addressed chunk with a random payload of 1 to
// swarm.ChunkSize bytes.
func randomCAC(rnd *rand.Rand) (chunk, error) {
	ch, err := randomCACChunk(rnd)
	if err != nil {
		return chunk{}, err
	}
	return chunk{addr: ch.Address(), data: ch.Data()}, nil
}

// randomSOC returns a single-owner chunk with a random id and payload, signed
// by a fresh key.
func randomSOC(rnd *rand.Rand) (chunk, error) {
	ch, err := randomCACChunk(rnd)
	if err != nil {
		return chunk{}, err
	}
	id := make([]byte, swarm.HashSize)
	if _, err := rnd.Read(id); err != nil {
		return chunk{}, fmt.Errorf("create soc id: %w", err)
	}
	key, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return chunk{}, fmt.Errorf("create key: %w", err)
	}
	sch, err := soc.New(id, ch).Sign(crypto.NewDefaultSigner(key))
	if err != nil {
		return chunk{}, fmt.Errorf("sign soc: %w", err)
	}
	return chunk{addr: sch.Address(), data: sch.Data()}, nil
}

func randomCACChunk(rnd *rand.Rand) (swarm.Chunk, error) {
	payload := make([]byte, 1+rnd.Intn(swarm.ChunkSize))
	if _, err := rnd.Read(payload); err != nil {
		return nil, fmt.Errorf("create payload: %w", err)
	}
	ch, err := cac.New(payload)
	if err != nil {
		return nil, fmt.Errorf("create cac: %w", err)
	}
	return ch, nil
}
