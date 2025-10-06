package soc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	GasPrice       string
	PostageTTL     time.Duration
	PostageDepth   uint64
	PostageLabel   string
	RequestTimeout time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GasPrice:       "",
		PostageTTL:     24 * time.Hour,
		PostageDepth:   16,
		PostageLabel:   "test-label",
		RequestTimeout: 5 * time.Minute,
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
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	payload := []byte("Hello Swarm :)")

	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return fmt.Errorf("generate secp256k1 key: %w", err)
	}
	signer := crypto.NewDefaultSigner(privKey)

	ch, err := cac.New(payload)
	if err != nil {
		return fmt.Errorf("create cac chunk: %w", err)
	}

	idBytes, err := randomID()
	if err != nil {
		return fmt.Errorf("random id: %w", err)
	}

	sch, err := soc.New(idBytes, ch).Sign(signer)
	if err != nil {
		return fmt.Errorf("sign soc chunk: %w", err)
	}

	chunkData := sch.Data()
	signatureBytes := chunkData[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]

	publicKey, err := signer.PublicKey()
	if err != nil {
		return fmt.Errorf("get public key: %w", err)
	}

	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return fmt.Errorf("get ethereum address: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, o.RequestTimeout)
	defer cancel()

	nodes, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("shuffled full node clients: %w", err)
	}

	if len(nodes) < 1 {
		return fmt.Errorf("soc test requires at least 1 full node")
	}

	node := nodes[0]
	nodeName := node.Name()
	c.logger.Infof("using node %s for soc test", nodeName)

	owner := hex.EncodeToString(ownerBytes)
	id := hex.EncodeToString(idBytes)
	sig := hex.EncodeToString(signatureBytes)

	batchID, err := node.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", nodeName, err)
	}

	c.logger.Infof("node %s: batch id %s", nodeName, batchID)
	c.logger.Infof("soc: submitting soc chunk %s to node %s", sch.Address().String(), nodeName)
	c.logger.Infof("soc: owner %s", owner)
	c.logger.Infof("soc: id %s", id)
	c.logger.Infof("soc: sig %s", sig)

	ref, err := node.UploadSOC(ctx, owner, id, sig, ch.Data(), batchID)
	if err != nil {
		return fmt.Errorf("node %s: upload soc chunk %w", nodeName, err)
	}

	c.logger.Infof("soc: chunk uploaded to node %s", nodeName)

	retrieved, err := node.DownloadChunk(ctx, ref, "", nil)
	if err != nil {
		return fmt.Errorf("node %s: download soc chunk %w", nodeName, err)
	}

	c.logger.Infof("soc: chunk retrieved from node %s", nodeName)

	if !bytes.Equal(retrieved, chunkData) {
		return errors.New("soc: retrieved chunk data does NOT match soc chunk")
	}

	return nil
}

func randomID() ([]byte, error) {
	key := make([]byte, 32)

	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}
