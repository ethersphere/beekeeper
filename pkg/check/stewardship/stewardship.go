// Package stewardship verifies that PUT /stewardship/{reference} re-seeds
// content to the network, including the dispersed replicas of root chunks.
package stewardship

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/replicas"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/storage"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options.
type Options struct {
	DataSize       int
	GasPrice       string
	PostageTTL     time.Duration
	PostageDepth   uint64
	PostageLabel   string
	RequestTimeout time.Duration
	RLevel         int
	Seed           int64
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		DataSize:       32 * 1024,
		GasPrice:       "",
		PostageTTL:     24 * time.Hour,
		PostageDepth:   22,
		PostageLabel:   "test-label",
		RequestTimeout: 5 * time.Minute,
		RLevel:         int(redundancy.MEDIUM),
		Seed:           0,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{logger: logger}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rLevel := redundancy.Level(o.RLevel)
	if rLevel == redundancy.NONE {
		return fmt.Errorf("stewardship check requires a non-zero redundancy level")
	}

	ctx, cancel := context.WithTimeout(ctx, o.RequestTimeout)
	defer cancel()

	rnd := random.PseudoGenerator(o.Seed)
	nodes, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("shuffled full node clients: %w", err)
	}
	if len(nodes) < 2 {
		return fmt.Errorf("stewardship check requires at least 2 full nodes, got %d", len(nodes))
	}

	uploader := nodes[0]
	batchID, err := uploader.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch: %w", uploader.Name(), err)
	}
	c.logger.Infof("stewardship: uploader %s, batch %s, rLevel %v", uploader.Name(), batchID, rLevel)

	state := &env{nodes: nodes, uploader: uploader, batchID: batchID, rLevel: rLevel, opts: o, rnd: rnd}

	// Every scenario runs even if an earlier one fails, so a single run reports
	// everything that is broken rather than only the first thing.
	var errs []error
	for _, sc := range []struct {
		name string
		fn   func(*Check, context.Context, *env) error
	}{
		{"root replicas restored", (*Check).checkRootReplicas},
		{"encrypted reference", (*Check).checkEncryptedReference},
		{"non-CAC (SOC) root accepted", (*Check).checkSOCRoot},
		{"per-file root replicas restored", (*Check).checkPerFileReplicas},
	} {
		c.logger.Infof("stewardship: --- %s ---", sc.name)
		if err := sc.fn(c, ctx, state); err != nil {
			c.logger.Errorf("stewardship: %s FAILED: %v", sc.name, err)
			errs = append(errs, fmt.Errorf("%s: %w", sc.name, err))
			continue
		}
		c.logger.Infof("stewardship: %s OK", sc.name)
	}

	return errors.Join(errs...)
}

type env struct {
	nodes    []*bee.Client
	uploader *bee.Client
	batchID  string
	rLevel   redundancy.Level
	opts     Options
	rnd      *rand.Rand
}

// checkRootReplicas uploads content with no redundancy, confirms the dispersed
// replicas of its root chunk are absent, then reuploads through stewardship at
// the configured level and requires every replica to be present and to wrap the
// root chunk byte for byte.
func (c *Check) checkRootReplicas(ctx context.Context, e *env) error {
	none := redundancy.NONE
	data := randomData(e.rnd, e.opts.DataSize)

	ref, err := e.uploader.UploadBytes(ctx, data, api.UploadOptions{BatchID: e.batchID, RLevel: &none})
	if err != nil {
		return fmt.Errorf("upload bytes: %w", err)
	}
	c.logger.Infof("stewardship: root %s", ref)

	reps, err := replicaAddresses(ref, e.rLevel)
	if err != nil {
		return err
	}

	present, err := e.countPresent(ctx, reps)
	if err != nil {
		return err
	}
	if present != 0 {
		return fmt.Errorf("expected 0 replicas before reupload, found %d", present)
	}

	if err := e.uploader.Reupload(ctx, ref, api.StewardshipOptions{BatchID: e.batchID, RLevel: &e.rLevel}); err != nil {
		return fmt.Errorf("reupload: %w", err)
	}

	present, err = e.countPresent(ctx, reps)
	if err != nil {
		return err
	}
	if present != len(reps) {
		return fmt.Errorf("expected %d replicas after reupload, found %d", len(reps), present)
	}

	return e.verifyReplicasWrap(ctx, ref, reps)
}

// checkEncryptedReference guards the encrypted-reference path: the 64-byte
// reference must be trimmed to its 32-byte content address before the root
// chunk is looked up, otherwise the reupload fails outright.
func (c *Check) checkEncryptedReference(ctx context.Context, e *env) error {
	none := redundancy.NONE
	data := randomData(e.rnd, e.opts.DataSize)

	ref, err := e.uploader.UploadBytes(ctx, data, api.UploadOptions{BatchID: e.batchID, RLevel: &none, Encrypt: true})
	if err != nil {
		return fmt.Errorf("upload encrypted bytes: %w", err)
	}
	if len(ref.Bytes()) != 2*swarm.HashSize {
		return fmt.Errorf("expected a %d-byte encrypted reference, got %d", 2*swarm.HashSize, len(ref.Bytes()))
	}
	contentAddr := swarm.NewAddress(ref.Bytes()[:swarm.HashSize])
	c.logger.Infof("stewardship: encrypted ref %s, content address %s", ref, contentAddr)

	if err := e.uploader.Reupload(ctx, ref, api.StewardshipOptions{BatchID: e.batchID, RLevel: &e.rLevel}); err != nil {
		return fmt.Errorf("reupload encrypted reference: %w", err)
	}

	reps, err := replicaAddresses(contentAddr, e.rLevel)
	if err != nil {
		return err
	}
	present, err := e.countPresent(ctx, reps)
	if err != nil {
		return err
	}
	if present != len(reps) {
		return fmt.Errorf("expected %d replicas of the trimmed content address, found %d", len(reps), present)
	}

	got, err := e.uploader.DownloadBytes(ctx, ref, nil)
	if err != nil {
		return fmt.Errorf("download encrypted bytes: %w", err)
	}
	if !bytes.Equal(got, data) {
		return errors.New("downloaded encrypted content does not match the original")
	}
	return nil
}

// checkSOCRoot reuploads a single owner chunk reference. traversal.Traverse
// supports SOC roots and the API documents the reference as "any type", so a
// SOC must not be rejected - in particular not on the default code path, where
// no Swarm-Redundancy-Level header is sent and the node falls back to
// redundancy.DefaultUploadLevel.
func (c *Check) checkSOCRoot(ctx context.Context, e *env) error {
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	signer := crypto.NewDefaultSigner(privKey)

	ch, err := cac.New(randomData(e.rnd, 64))
	if err != nil {
		return fmt.Errorf("create cac: %w", err)
	}
	id := randomData(e.rnd, swarm.HashSize)
	sch, err := soc.New(id, ch).Sign(signer)
	if err != nil {
		return fmt.Errorf("sign soc: %w", err)
	}
	owner, err := signer.EthereumAddress()
	if err != nil {
		return fmt.Errorf("owner address: %w", err)
	}
	sig := sch.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]

	ref, err := e.uploader.UploadSOC(ctx, hex.EncodeToString(owner.Bytes()), hex.EncodeToString(id), hex.EncodeToString(sig), ch.Data(), e.batchID)
	if err != nil {
		return fmt.Errorf("upload soc: %w", err)
	}
	c.logger.Infof("stewardship: soc %s", ref)

	// No RLevel: exercises the default path a client hits when it omits the header.
	if err := e.uploader.Reupload(ctx, ref, api.StewardshipOptions{BatchID: e.batchID}); err != nil {
		return fmt.Errorf("reupload soc reference (no redundancy header): %w", err)
	}

	if err := e.uploader.Reupload(ctx, ref, api.StewardshipOptions{BatchID: e.batchID, RLevel: &e.rLevel}); err != nil {
		return fmt.Errorf("reupload soc reference (rLevel %v): %w", e.rLevel, err)
	}
	return nil
}

// checkPerFileReplicas covers a /bzz upload, where the file and the manifest go
// through separate pipelines and each gets dispersed replicas of its own root
// chunk. Reuploading the manifest reference must restore both sets, otherwise a
// GET /bzz/{ref}/{path} - which joins the *file* reference through the replicas
// getter - is left with no replica fallback.
func (c *Check) checkPerFileReplicas(ctx context.Context, e *env) error {
	none := redundancy.NONE
	data := randomData(e.rnd, e.opts.DataSize)

	// The file pipeline is deterministic, so uploading the same bytes through
	// /bytes at the same level yields the file root that /bzz embeds.
	fileRoot, err := e.uploader.UploadBytes(ctx, data, api.UploadOptions{BatchID: e.batchID, RLevel: &none})
	if err != nil {
		return fmt.Errorf("upload file bytes: %w", err)
	}

	f := bee.NewBufferFile("stewardship.bin", bytes.NewBuffer(data))
	if err := e.uploader.UploadFile(ctx, &f, api.UploadOptions{BatchID: e.batchID, RLevel: &none}); err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	manifestRoot := f.Address()
	c.logger.Infof("stewardship: manifest root %s, file root %s", manifestRoot, fileRoot)

	if err := e.uploader.Reupload(ctx, manifestRoot, api.StewardshipOptions{BatchID: e.batchID, RLevel: &e.rLevel}); err != nil {
		return fmt.Errorf("reupload manifest: %w", err)
	}

	manifestReps, err := replicaAddresses(manifestRoot, e.rLevel)
	if err != nil {
		return err
	}
	present, err := e.countPresent(ctx, manifestReps)
	if err != nil {
		return err
	}
	if present != len(manifestReps) {
		return fmt.Errorf("manifest root: expected %d replicas, found %d", len(manifestReps), present)
	}

	fileReps, err := replicaAddresses(fileRoot, e.rLevel)
	if err != nil {
		return err
	}
	present, err = e.countPresent(ctx, fileReps)
	if err != nil {
		return err
	}
	if present != len(fileReps) {
		return fmt.Errorf("file root %s: expected %d replicas after reuploading manifest %s, found %d "+
			"(GET /bzz/{ref}/{path} joins the file reference through the replicas getter, so these are the ones a download falls back on)",
			fileRoot, len(fileReps), manifestRoot, present)
	}
	return nil
}

// countPresent reports how many of the given chunks at least one full node
// holds locally. HEAD /chunks/{addr} is a local-store lookup, so this does not
// pay the retrieval timeout for chunks that are genuinely absent.
func (e *env) countPresent(ctx context.Context, addrs []swarm.Address) (int, error) {
	count := 0
	for _, addr := range addrs {
		found := false
		for _, n := range e.nodes {
			has, err := n.HasChunkLocal(ctx, addr)
			if err != nil {
				return 0, fmt.Errorf("node %s: has chunk %s: %w", n.Name(), addr, err)
			}
			if has {
				found = true
				break
			}
		}
		if found {
			count++
		}
	}
	return count, nil
}

// verifyReplicasWrap requires each replica to be a valid SOC whose wrapped
// chunk is the root chunk, which is exactly what replicas.getter unwraps at
// download time. A replica that is merely present but wraps the wrong chunk
// would be useless as a retrieval fallback.
func (e *env) verifyReplicasWrap(ctx context.Context, root swarm.Address, reps []swarm.Address) error {
	rootData, err := e.uploader.DownloadChunk(ctx, root, "", nil)
	if err != nil {
		return fmt.Errorf("download root chunk %s: %w", root, err)
	}
	for _, addr := range reps {
		data, err := e.uploader.DownloadChunk(ctx, addr, "", nil)
		if err != nil {
			return fmt.Errorf("download replica %s: %w", addr, err)
		}
		sch, err := soc.FromChunk(swarm.NewChunk(addr, data))
		if err != nil {
			return fmt.Errorf("replica %s is not a valid soc: %w", addr, err)
		}
		if !bytes.Equal(sch.WrappedChunk().Data(), rootData) {
			return fmt.Errorf("replica %s wraps %s, want root %s", addr, sch.WrappedChunk().Address(), root)
		}
	}
	return nil
}

// replicaAddresses returns the dispersed replica addresses for a root address,
// derived through replicas.NewPutter so the enumeration matches the node's.
func replicaAddresses(root swarm.Address, rLevel redundancy.Level) ([]swarm.Address, error) {
	addr := root
	if len(addr.Bytes()) > swarm.HashSize {
		addr = swarm.NewAddress(addr.Bytes()[:swarm.HashSize])
	}

	var (
		mu   sync.Mutex
		out  []swarm.Address
		ch   = swarm.NewChunk(addr, make([]byte, swarm.ChunkWithSpanSize))
		sink = storage.PutterFunc(func(_ context.Context, c swarm.Chunk) error {
			mu.Lock()
			defer mu.Unlock()
			out = append(out, c.Address())
			return nil
		})
	)
	if err := replicas.NewPutter(sink, rLevel).Put(context.Background(), ch); err != nil {
		return nil, fmt.Errorf("derive replica addresses for %s: %w", root, err)
	}
	if len(out) != rLevel.GetReplicaCount() {
		return nil, fmt.Errorf("derived %d replica addresses, want %d", len(out), rLevel.GetReplicaCount())
	}
	return out, nil
}

func randomData(rnd *rand.Rand, n int) []byte {
	b := make([]byte, n)
	_, _ = rnd.Read(b)
	return b
}
