package chunkconvergence

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/postage"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

const bucketDepth = postage.BucketDepth

// Options represents check options.
type Options struct {
	PostageTTL     time.Duration
	PostageDepth   uint64
	PostageLabel   string
	SyncWait       time.Duration
	RequestTimeout time.Duration
	Password       string // password for decrypting the batch-issuer Swarm key
}

// NewDefaultOptions returns new default options.
func NewDefaultOptions() Options {
	return Options{
		PostageTTL:     24 * time.Hour,
		PostageDepth:   21,
		PostageLabel:   "chunk-convergence",
		SyncWait:       45 * time.Second,
		RequestTimeout: 10 * time.Minute,
		Password:       "beekeeper",
	}
}

var _ beekeeper.Action = (*Check)(nil)

// Check verifies pullsync convergence for conflicting SOC/CAC stamp claims.
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

	ctx, cancel := context.WithTimeout(ctx, o.RequestTimeout)
	defer cancel()

	fullNodes, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("full node clients: %w", err)
	}
	if len(fullNodes) < 2 {
		return fmt.Errorf("chunk convergence requires at least 2 full nodes, got %d", len(fullNodes))
	}

	n1, n2, err := pickClosestPair(ctx, fullNodes)
	if err != nil {
		return err
	}
	c.logger.Infof("neighborhood upload pair: %s and %s", n1.Name(), n2.Name())
	neighborhood := []*bee.Client{n1, n2}

	issuer, signer, batchID, err := c.setupIssuerBatch(ctx, cluster, o)
	if err != nil {
		return err
	}
	c.logger.Infof("batch issuer=%s batch_id=%s", issuer.Name(), batchID)

	scenarios := []struct {
		name string
		run  func() error
	}{
		{"1 divergent soc identical stamp", func() error {
			return c.scenarioDivergentSOC(ctx, issuer, neighborhood, batchID, o.SyncWait)
		}},
		{"2 soc same index higher timestamp", func() error {
			return c.scenarioSOCSameIndexTimestamp(ctx, issuer, signer, neighborhood, batchID, o.SyncWait)
		}},
		{"3 soc different index higher timestamp", func() error {
			return c.scenarioSOCDifferentIndexTimestamp(ctx, issuer, neighborhood, batchID, o.SyncWait)
		}},
		{"4 cac same index higher timestamp", func() error {
			return c.scenarioCACSameIndexTimestamp(ctx, issuer, signer, neighborhood, batchID, o.SyncWait)
		}},
		{"5 cac same index equal timestamp", func() error {
			return c.scenarioCACSameIndexEqualTimestamp(ctx, issuer, signer, neighborhood, batchID, o.SyncWait)
		}},
		{"6 soc stale stamp redelivery race", func() error {
			return c.scenarioSOCStaleStampRedeliveryRace(ctx, issuer, signer, neighborhood, batchID, o, o.SyncWait)
		}},
	}

	var errs error
	for _, s := range scenarios {
		c.logger.Infof("========== %s ==========", s.name)
		if err := s.run(); err != nil {
			c.logger.Errorf("%s failed: %v", s.name, err)
			errs = errors.Join(errs, fmt.Errorf("%s: %w", s.name, err))
			continue
		}
		c.logger.Infof("%s passed", s.name)
	}
	return errs
}

func (c *Check) setupIssuerBatch(ctx context.Context, cluster orchestration.Cluster, o Options) (*bee.Client, crypto.Signer, string, error) {
	nodes := cluster.Nodes()
	var issuerNode orchestration.Node
	var issuerName string
	for name, n := range nodes {
		if n.SwarmKey() != nil {
			issuerNode = n
			issuerName = name
			break
		}
	}
	if issuerNode == nil {
		return nil, nil, "", fmt.Errorf("no node with swarm-key found; bootnode with configured swarm-key is required")
	}

	privKey, err := issuerNode.SwarmKey().Decrypt(o.Password)
	if err != nil {
		return nil, nil, "", fmt.Errorf("decrypt swarm key for %s: %w", issuerName, err)
	}
	signer := crypto.NewDefaultSigner(privKey)

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, nil, "", fmt.Errorf("nodes clients: %w", err)
	}
	issuer, ok := clients[issuerName]
	if !ok {
		return nil, nil, "", fmt.Errorf("client for issuer node %s not found", issuerName)
	}

	label := fmt.Sprintf("%s-%s", o.PostageLabel, hex.EncodeToString(randomBytes(4)))
	batchID, err := issuer.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, label)
	if err != nil {
		return nil, nil, "", fmt.Errorf("create batch on %s: %w", issuerName, err)
	}
	return issuer, signer, batchID, nil
}

func (c *Check) scenarioDivergentSOC(ctx context.Context, issuer *bee.Client, neighborhood []*bee.Client, batchID string, syncWait time.Duration) error {
	signer, owner, id, addr, err := mineSOC(ctx, neighborhood[0], neighborhood[1])
	if err != nil {
		return err
	}
	low, high, err := orderedSOCs(signer, id)
	if err != nil {
		return err
	}
	stamp, err := issuer.CreateEnvelope(ctx, addr, batchID)
	if err != nil {
		return fmt.Errorf("envelope: %w", err)
	}
	stampBytes, err := stamp.MarshalBinary()
	if err != nil {
		return err
	}

	c.logger.Infof("soc=%s wrapped_low=%s wrapped_high=%s", addr, mustWrapped(low), mustWrapped(high))
	// Identical stamp: second upload may lose the wrapped-CAC tie-break if the
	// winner already synced. Require only the first (winning) upload.
	if err := uploadSOCPair(ctx, neighborhood[0], neighborhood[1], owner, id, low, high, stampBytes, stampBytes, false); err != nil {
		return err
	}
	time.Sleep(syncWait)
	return assertNeighborhoodPayload(ctx, c.logger, neighborhood, addr, low.Data())
}

func (c *Check) scenarioSOCSameIndexTimestamp(ctx context.Context, issuer *bee.Client, batchSigner crypto.Signer, neighborhood []*bee.Client, batchID string, syncWait time.Duration) error {
	signer, owner, id, addr, err := mineSOC(ctx, neighborhood[0], neighborhood[1])
	if err != nil {
		return err
	}
	older, err := buildSOC(signer, id, append([]byte("soc-same-idx-older-"), randomBytes(4)...))
	if err != nil {
		return err
	}
	newer, err := buildSOC(signer, id, append([]byte("soc-same-idx-newer-"), randomBytes(4)...))
	if err != nil {
		return err
	}

	batchBytes, err := hex.DecodeString(batchID)
	if err != nil {
		return err
	}
	// CreateEnvelope yields a postage-valid index; reuse it with two timestamps.
	env, err := issuer.CreateEnvelope(ctx, addr, batchID)
	if err != nil {
		return fmt.Errorf("envelope for index: %w", err)
	}
	index := append([]byte{}, env.Index()...)
	ts := binary.BigEndian.Uint64(env.Timestamp())
	stampOld, err := signStamp(batchSigner, batchBytes, index, uint64ToBytes(ts), addr)
	if err != nil {
		return fmt.Errorf("sign older stamp: %w", err)
	}
	stampNew, err := signStamp(batchSigner, batchBytes, index, uint64ToBytes(ts+1), addr)
	if err != nil {
		return fmt.Errorf("sign newer stamp: %w", err)
	}
	older = older.WithStamp(stampOld)
	newer = newer.WithStamp(stampNew)
	oldBytes, _ := stampOld.MarshalBinary()
	newBytes, _ := stampNew.MarshalBinary()

	c.logger.Infof("soc=%s index=%s", addr, hex.EncodeToString(index))
	// Older first, then newer — avoids racing a losing upload after the winner synced.
	if err := uploadSOCPair(ctx, neighborhood[0], neighborhood[1], owner, id, older, newer, oldBytes, newBytes, true); err != nil {
		return err
	}
	time.Sleep(syncWait)
	return assertNeighborhoodPayload(ctx, c.logger, neighborhood, addr, newer.Data())
}

func (c *Check) scenarioSOCDifferentIndexTimestamp(ctx context.Context, issuer *bee.Client, neighborhood []*bee.Client, batchID string, syncWait time.Duration) error {
	signer, owner, id, addr, err := mineSOC(ctx, neighborhood[0], neighborhood[1])
	if err != nil {
		return err
	}
	older, err := buildSOC(signer, id, append([]byte("soc-diff-idx-older-"), randomBytes(4)...))
	if err != nil {
		return err
	}
	newer, err := buildSOC(signer, id, append([]byte("soc-diff-idx-newer-"), randomBytes(4)...))
	if err != nil {
		return err
	}

	stampOld, err := issuer.CreateEnvelope(ctx, addr, batchID)
	if err != nil {
		return fmt.Errorf("envelope older: %w", err)
	}
	stampNew, err := issuer.CreateEnvelope(ctx, addr, batchID)
	if err != nil {
		return fmt.Errorf("envelope newer: %w", err)
	}
	for binary.BigEndian.Uint64(stampNew.Timestamp()) <= binary.BigEndian.Uint64(stampOld.Timestamp()) {
		stampNew, err = issuer.CreateEnvelope(ctx, addr, batchID)
		if err != nil {
			return fmt.Errorf("envelope newer retry: %w", err)
		}
	}
	older = older.WithStamp(stampOld)
	newer = newer.WithStamp(stampNew)
	oldBytes, _ := stampOld.MarshalBinary()
	newBytes, _ := stampNew.MarshalBinary()

	c.logger.Infof("soc=%s idx_old=%s idx_new=%s", addr, hex.EncodeToString(stampOld.Index()), hex.EncodeToString(stampNew.Index()))
	if err := uploadSOCPair(ctx, neighborhood[0], neighborhood[1], owner, id, older, newer, oldBytes, newBytes, true); err != nil {
		return err
	}
	time.Sleep(syncWait)
	return assertNeighborhoodPayload(ctx, c.logger, neighborhood, addr, newer.Data())
}

func (c *Check) scenarioCACSameIndexTimestamp(ctx context.Context, issuer *bee.Client, batchSigner crypto.Signer, neighborhood []*bee.Client, batchID string, syncWait time.Duration) error {
	addrs, err := neighborhood[0].Addresses(ctx)
	if err != nil {
		return err
	}
	low, high, err := orderedCACsInBucket(addrs.Overlay)
	if err != nil {
		return err
	}
	batchBytes, err := hex.DecodeString(batchID)
	if err != nil {
		return err
	}
	// Valid index for this postage bucket (both CACs share the bucket).
	env, err := issuer.CreateEnvelope(ctx, low.Address(), batchID)
	if err != nil {
		return fmt.Errorf("envelope for index: %w", err)
	}
	index := append([]byte{}, env.Index()...)
	ts := binary.BigEndian.Uint64(env.Timestamp())
	// higher timestamp on high address chunk so timestamp rule (not address tie-break) decides
	stampLow, err := signStamp(batchSigner, batchBytes, index, uint64ToBytes(ts), low.Address())
	if err != nil {
		return err
	}
	stampHigh, err := signStamp(batchSigner, batchBytes, index, uint64ToBytes(ts+1), high.Address())
	if err != nil {
		return err
	}
	low = low.WithStamp(stampLow)
	high = high.WithStamp(stampHigh)

	c.logger.Infof("cac_low=%s cac_high=%s index=%s", low.Address(), high.Address(), hex.EncodeToString(index))
	if err := uploadChunkPair(ctx, neighborhood[0], neighborhood[1], low, high); err != nil {
		return err
	}
	time.Sleep(syncWait)

	for _, n := range neighborhood {
		hasWant, err := n.HasChunk(ctx, high.Address())
		if err != nil {
			return err
		}
		hasLose, err := n.HasChunk(ctx, low.Address())
		if err != nil {
			return err
		}
		c.logger.Infof("%s has_winner=%v has_loser=%v", n.Name(), hasWant, hasLose)
		if !hasWant {
			return fmt.Errorf("node %s missing higher-timestamp CAC %s", n.Name(), high.Address())
		}
		if hasLose {
			return fmt.Errorf("node %s still has lower-timestamp CAC %s", n.Name(), low.Address())
		}
	}
	return nil
}

func (c *Check) scenarioCACSameIndexEqualTimestamp(ctx context.Context, issuer *bee.Client, batchSigner crypto.Signer, neighborhood []*bee.Client, batchID string, syncWait time.Duration) error {
	addrs, err := neighborhood[0].Addresses(ctx)
	if err != nil {
		return err
	}
	low, high, err := orderedCACsInBucket(addrs.Overlay)
	if err != nil {
		return err
	}
	batchBytes, err := hex.DecodeString(batchID)
	if err != nil {
		return err
	}
	// Same batch, stamp index and timestamp; signatures differ (each stamp is
	// bound to its CAC address). At equal timestamp the lower address wins.
	env, err := issuer.CreateEnvelope(ctx, low.Address(), batchID)
	if err != nil {
		return fmt.Errorf("envelope for index: %w", err)
	}
	index := append([]byte{}, env.Index()...)
	ts := uint64ToBytes(binary.BigEndian.Uint64(env.Timestamp()))

	stampLow, err := signStamp(batchSigner, batchBytes, index, ts, low.Address())
	if err != nil {
		return err
	}
	stampHigh, err := signStamp(batchSigner, batchBytes, index, ts, high.Address())
	if err != nil {
		return err
	}
	low = low.WithStamp(stampLow)
	high = high.WithStamp(stampHigh)

	c.logger.Infof("cac_low=%s cac_high=%s index=%s equal_ts address tie-break", low.Address(), high.Address(), hex.EncodeToString(index))
	// Upload loser (higher address) first, then winner (lower address).
	if err := uploadChunkPair(ctx, neighborhood[0], neighborhood[1], high, low); err != nil {
		return err
	}
	time.Sleep(syncWait)

	for _, n := range neighborhood {
		hasWant, err := n.HasChunk(ctx, low.Address())
		if err != nil {
			return err
		}
		hasLose, err := n.HasChunk(ctx, high.Address())
		if err != nil {
			return err
		}
		c.logger.Infof("%s has_winner=%v has_loser=%v", n.Name(), hasWant, hasLose)
		if !hasWant {
			return fmt.Errorf("node %s missing lower-address CAC %s", n.Name(), low.Address())
		}
		if hasLose {
			return fmt.Errorf("node %s still has higher-address CAC %s", n.Name(), high.Address())
		}
	}
	return nil
}

// scenarioSOCStaleStampRedeliveryRace races three divergent SOCs at one address:
//
//	SOC₁ under stamp S (batch A, lower timestamp) → uploaded to Sam
//	SOC₂ under stamp T (batch B, higher timestamp) → uploaded to Sara
//	SOC₃ reuses stamp S with payload P₃ whose wrapped CAC beats P₁ and P₂ →
//	uploaded to both nodes in parallel with the SOC₁/SOC₂ uploads
//
// After sync, every neighborhood node must hold P₃ (wrapped-CAC winner under
// the reused stamp S / resolveDivergence).
func (c *Check) scenarioSOCStaleStampRedeliveryRace(ctx context.Context, issuer *bee.Client, batchSigner crypto.Signer, neighborhood []*bee.Client, batchID string, o Options, syncWait time.Duration) error {
	sam, sara := neighborhood[0], neighborhood[1]

	batchB, err := issuer.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel+"-stale-"+hex.EncodeToString(randomBytes(3)))
	if err != nil {
		return fmt.Errorf("batch B: %w", err)
	}

	signer, owner, id, addr, err := mineSOC(ctx, sam, sara)
	if err != nil {
		return err
	}
	soc1, soc2, soc3, err := orderedStaleRaceSOCs(signer, id)
	if err != nil {
		return err
	}

	batchABytes, err := hex.DecodeString(batchID)
	if err != nil {
		return err
	}
	batchBBytes, err := hex.DecodeString(batchB)
	if err != nil {
		return err
	}

	envS, err := issuer.CreateEnvelope(ctx, addr, batchID)
	if err != nil {
		return fmt.Errorf("envelope stamp S: %w", err)
	}
	envT, err := issuer.CreateEnvelope(ctx, addr, batchB)
	if err != nil {
		return fmt.Errorf("envelope stamp T: %w", err)
	}
	tsS := binary.BigEndian.Uint64(envS.Timestamp())
	tsT := binary.BigEndian.Uint64(envT.Timestamp())
	if tsS <= tsT {
		tsS = tsT + 1
	}
	stampS, err := signStamp(batchSigner, batchABytes, append([]byte{}, envS.Index()...), uint64ToBytes(tsS), addr)
	if err != nil {
		return fmt.Errorf("sign stamp S: %w", err)
	}
	stampT, err := signStamp(batchSigner, batchBBytes, append([]byte{}, envT.Index()...), uint64ToBytes(tsT), addr)
	if err != nil {
		return fmt.Errorf("sign stamp T: %w", err)
	}
	stampSBytes, err := stampS.MarshalBinary()
	if err != nil {
		return err
	}
	stampTBytes, err := stampT.MarshalBinary()
	if err != nil {
		return err
	}

	c.logger.Infof(
		"soc=%s stale-stamp race sam=%s sara=%s wrapped_p1=%s wrapped_p2=%s wrapped_p3=%s batchA=%s… batchB=%s… tsS=%d tsT=%d",
		addr, sam.Name(), sara.Name(), mustWrapped(soc1), mustWrapped(soc2), mustWrapped(soc3),
		batchID[:8], batchB[:8], tsS, tsT,
	)

	type uploadResult struct {
		label string
		err   error
	}
	results := make(chan uploadResult, 4)

	go func() {
		results <- uploadResult{"SOC1→" + sam.Name(), uploadSOC(ctx, sam, owner, id, soc1, stampSBytes)}
	}()
	go func() {
		results <- uploadResult{"SOC2→" + sara.Name(), uploadSOC(ctx, sara, owner, id, soc2, stampTBytes)}
	}()
	go func() {
		results <- uploadResult{"SOC3→" + sam.Name(), uploadSOC(ctx, sam, owner, id, soc3, stampSBytes)}
	}()
	go func() {
		results <- uploadResult{"SOC3→" + sara.Name(), uploadSOC(ctx, sara, owner, id, soc3, stampSBytes)}
	}()

	for range 4 {
		r := <-results
		if r.err != nil {
			c.logger.Infof("upload %s returned: %v", r.label, r.err)
		} else {
			c.logger.Infof("upload %s ok", r.label)
		}
	}
	c.logger.Infof("race uploads finished; waiting %s for neighborhood to settle on P3", syncWait)

	time.Sleep(syncWait)

	for _, n := range neighborhood {
		got, err := n.DownloadChunk(ctx, addr, "", nil)
		if err != nil {
			c.logger.Errorf("%s download after race: %v", n.Name(), err)
			continue
		}
		switch {
		case bytes.Equal(got, soc3.Data()):
			c.logger.Infof("%s holds P3 — expected", n.Name())
		case bytes.Equal(got, soc2.Data()):
			c.logger.Errorf("%s holds P2 (stamp T) — want P3", n.Name())
		case bytes.Equal(got, soc1.Data()):
			c.logger.Errorf("%s holds P1 — want P3", n.Name())
		default:
			c.logger.Errorf("%s holds unexpected payload len=%d", n.Name(), len(got))
		}
	}

	if err := assertNeighborhoodPayload(ctx, c.logger, neighborhood, addr, soc3.Data()); err != nil {
		return fmt.Errorf("after stale-stamp race (want P3 on both): %w", err)
	}
	return nil
}

// orderedStaleRaceSOCs builds three SOCs at the same address where P₃'s wrapped
// CAC is lexicographically lower than both P₁ and P₂, so resolveDivergence would
// accept P₃ over either stored payload when the stale-stamp guard is absent.
func orderedStaleRaceSOCs(signer crypto.Signer, id []byte) (p1, p2, p3 swarm.Chunk, err error) {
	for range 10_000 {
		p1, err = buildSOC(signer, id, append([]byte("stale-p1-"), randomBytes(8)...))
		if err != nil {
			return nil, nil, nil, err
		}
		p2, err = buildSOC(signer, id, append([]byte("stale-p2-"), randomBytes(8)...))
		if err != nil {
			return nil, nil, nil, err
		}
		p3, err = buildSOC(signer, id, append([]byte("stale-p3-"), randomBytes(8)...))
		if err != nil {
			return nil, nil, nil, err
		}
		w1, w2, w3 := mustWrapped(p1).Bytes(), mustWrapped(p2).Bytes(), mustWrapped(p3).Bytes()
		if bytes.Compare(w3, w1) < 0 && bytes.Compare(w3, w2) < 0 &&
			!bytes.Equal(w1, w2) && !bytes.Equal(w1, w3) && !bytes.Equal(w2, w3) {
			return p1, p2, p3, nil
		}
	}
	return nil, nil, nil, fmt.Errorf("could not mine P1/P2/P3 with wrapped(P3) lower than both")
}

func uploadSOC(ctx context.Context, n *bee.Client, owner string, id []byte, ch swarm.Chunk, stamp []byte) error {
	sig := hex.EncodeToString(ch.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	wrapped := ch.Data()[swarm.HashSize+swarm.SocSignatureSize:]
	idHex := hex.EncodeToString(id)
	if _, err := n.UploadSOCWithStamp(ctx, owner, idHex, sig, wrapped, stamp); err != nil {
		return fmt.Errorf("upload to %s: %w", n.Name(), err)
	}
	return nil
}

// uploadSOCPair stores ch1 on n1 then ch2 on n2. Callers should pass the
// expected loser/older chunk first so the winner can overwrite during sync.
// If requireSecond is false, a failed second upload is ignored (tie-break loser).
func uploadSOCPair(ctx context.Context, n1, n2 *bee.Client, owner string, id []byte, ch1, ch2 swarm.Chunk, stamp1, stamp2 []byte, requireSecond bool) error {
	sig1 := hex.EncodeToString(ch1.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	sig2 := hex.EncodeToString(ch2.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	wrapped1 := ch1.Data()[swarm.HashSize+swarm.SocSignatureSize:]
	wrapped2 := ch2.Data()[swarm.HashSize+swarm.SocSignatureSize:]
	idHex := hex.EncodeToString(id)

	if _, err := n1.UploadSOCWithStamp(ctx, owner, idHex, sig1, wrapped1, stamp1); err != nil {
		return fmt.Errorf("upload to %s: %w", n1.Name(), err)
	}
	if _, err := n2.UploadSOCWithStamp(ctx, owner, idHex, sig2, wrapped2, stamp2); err != nil {
		if requireSecond {
			return fmt.Errorf("upload to %s: %w", n2.Name(), err)
		}
	}
	return nil
}

func uploadChunkPair(ctx context.Context, n1, n2 *bee.Client, ch1, ch2 swarm.Chunk) error {
	st1, err := ch1.Stamp().MarshalBinary()
	if err != nil {
		return err
	}
	st2, err := ch2.Stamp().MarshalBinary()
	if err != nil {
		return err
	}
	if _, err := n1.UploadChunk(ctx, ch1.Data(), api.UploadOptions{Stamp: hex.EncodeToString(st1), Direct: true}); err != nil {
		return fmt.Errorf("upload to %s: %w", n1.Name(), err)
	}
	if _, err := n2.UploadChunk(ctx, ch2.Data(), api.UploadOptions{Stamp: hex.EncodeToString(st2), Direct: true}); err != nil {
		return fmt.Errorf("upload to %s: %w", n2.Name(), err)
	}
	return nil
}

func assertNeighborhoodPayload(ctx context.Context, logger logging.Logger, nodes []*bee.Client, addr swarm.Address, want []byte) error {
	for _, n := range nodes {
		got, err := n.DownloadChunk(ctx, addr, "", nil)
		if err != nil {
			return fmt.Errorf("node %s download %s: %w", n.Name(), addr, err)
		}
		if !bytes.Equal(got, want) {
			logger.Errorf("node %s payload mismatch want_len=%d got_len=%d", n.Name(), len(want), len(got))
			return fmt.Errorf("node %s: payload mismatch", n.Name())
		}
	}
	return nil
}

func pickClosestPair(ctx context.Context, nodes orchestration.ClientList) (*bee.Client, *bee.Client, error) {
	overlays := make([]swarm.Address, len(nodes))
	for i, n := range nodes {
		a, err := n.Addresses(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("addresses for %s: %w", n.Name(), err)
		}
		overlays[i] = a.Overlay
	}
	bestI, bestJ, bestPO := 0, 1, -1
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			p := int(swarm.Proximity(overlays[i].Bytes(), overlays[j].Bytes()))
			if p > bestPO {
				bestPO, bestI, bestJ = p, i, j
			}
		}
	}
	if bestPO < 0 {
		return nil, nil, fmt.Errorf("could not find a node pair")
	}
	return nodes[bestI], nodes[bestJ], nil
}

func mineSOC(ctx context.Context, n1, n2 *bee.Client) (crypto.Signer, string, []byte, swarm.Address, error) {
	a1, err := n1.Addresses(ctx)
	if err != nil {
		return nil, "", nil, swarm.ZeroAddress, err
	}
	a2, err := n2.Addresses(ctx)
	if err != nil {
		return nil, "", nil, swarm.ZeroAddress, err
	}
	t1, t2 := a1.Overlay, a2.Overlay
	minPO := swarm.Proximity(t1.Bytes(), t2.Bytes())
	if minPO > 8 {
		minPO = 8
	}

	for {
		privKey, err := crypto.GenerateSecp256k1Key()
		if err != nil {
			return nil, "", nil, swarm.ZeroAddress, err
		}
		signer := crypto.NewDefaultSigner(privKey)
		ownerBytes, err := signer.EthereumAddress()
		if err != nil {
			return nil, "", nil, swarm.ZeroAddress, err
		}
		owner := hex.EncodeToString(ownerBytes.Bytes())
		for range 8192 {
			id := make([]byte, swarm.HashSize)
			if _, err := crand.Read(id); err != nil {
				return nil, "", nil, swarm.ZeroAddress, err
			}
			tmp, err := cac.New([]byte("x"))
			if err != nil {
				continue
			}
			ch, err := soc.New(id, tmp).Sign(signer)
			if err != nil {
				continue
			}
			a := ch.Address()
			if swarm.Proximity(a.Bytes(), t1.Bytes()) >= minPO &&
				swarm.Proximity(a.Bytes(), t2.Bytes()) >= minPO {
				return signer, owner, id, a, nil
			}
		}
	}
}

func orderedSOCs(signer crypto.Signer, id []byte) (low, high swarm.Chunk, err error) {
	a, err := buildSOC(signer, id, []byte("soc-conv-payload-aaa"))
	if err != nil {
		return nil, nil, err
	}
	b, err := buildSOC(signer, id, []byte("soc-conv-payload-bbb"))
	if err != nil {
		return nil, nil, err
	}
	wa, wb := mustWrapped(a), mustWrapped(b)
	if bytes.Compare(wa.Bytes(), wb.Bytes()) < 0 {
		return a, b, nil
	}
	return b, a, nil
}

func buildSOC(signer crypto.Signer, id, payload []byte) (swarm.Chunk, error) {
	ch, err := cac.New(payload)
	if err != nil {
		return nil, err
	}
	return soc.New(id, ch).Sign(signer)
}

func orderedCACsInBucket(target swarm.Address) (low, high swarm.Chunk, err error) {
	bucket := addressBucket(bucketDepth, target)
	a, err := cacInBucket(bucket, append([]byte("cac-conv-aaa-"), randomBytes(8)...))
	if err != nil {
		return nil, nil, err
	}
	b, err := cacInBucket(bucket, append([]byte("cac-conv-bbb-"), randomBytes(8)...))
	if err != nil {
		return nil, nil, err
	}
	for a.Address().Equal(b.Address()) {
		b, err = cacInBucket(bucket, randomBytes(32))
		if err != nil {
			return nil, nil, err
		}
	}
	if bytes.Compare(a.Address().Bytes(), b.Address().Bytes()) < 0 {
		return a, b, nil
	}
	return b, a, nil
}

func cacInBucket(bucket uint32, seed []byte) (swarm.Chunk, error) {
	for range 5_000_000 {
		payload := append(append([]byte{}, seed...), randomBytes(16)...)
		ch, err := cac.New(payload)
		if err != nil {
			return nil, err
		}
		if addressBucket(bucketDepth, ch.Address()) == bucket {
			return ch, nil
		}
		seed = randomBytes(16)
	}
	return nil, fmt.Errorf("could not mine CAC into bucket %d", bucket)
}

func addressBucket(depth uint8, addr swarm.Address) uint32 {
	return binary.BigEndian.Uint32(addr.Bytes()[:4]) >> (32 - depth)
}

func signStamp(signer crypto.Signer, batchID, index, ts []byte, addr swarm.Address) (*postage.Stamp, error) {
	digest, err := postage.ToSignDigest(addr.Bytes(), batchID, index, ts)
	if err != nil {
		return nil, err
	}
	sig, err := signer.Sign(digest)
	if err != nil {
		return nil, err
	}
	return postage.NewStamp(batchID, index, ts, sig), nil
}

func mustWrapped(ch swarm.Chunk) swarm.Address {
	s, err := soc.FromChunk(ch)
	if err != nil {
		return swarm.ZeroAddress
	}
	return s.WrappedChunk().Address()
}

func uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = crand.Read(b)
	return b
}
