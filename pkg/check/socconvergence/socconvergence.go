// Copyright 2026 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package socconvergence

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	GasPrice          string
	PostageTTL        time.Duration
	PostageDepth      uint64
	PostageLabel      string
	RequestTimeout    time.Duration
	SyncRetryInterval time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GasPrice:          "",
		PostageTTL:        24 * time.Hour,
		PostageDepth:      16,
		PostageLabel:      "soc-convergence-label",
		RequestTimeout:    5 * time.Minute,
		SyncRetryInterval: 2 * time.Second,
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	ctx, cancel := context.WithTimeout(ctx, o.RequestTimeout)
	defer cancel()

	rnd := random.PseudoGenerator(time.Now().UnixNano())
	fullNodeClients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("shuffled full node clients: %w", err)
	}

	if len(fullNodeClients) < 2 {
		return fmt.Errorf("socconvergence test requires at least 2 full nodes")
	}

	node0 := fullNodeClients[0]
	node1 := fullNodeClients[1]

	c.logger.Infof("socconvergence: using node0 '%s' and node1 '%s' (total full nodes: %d)",
		node0.Name(), node1.Name(), len(fullNodeClients))

	// Shared signer for same owner address
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return fmt.Errorf("generate secp256k1 key: %w", err)
	}
	signer := crypto.NewDefaultSigner(privKey)
	pubKey, err := signer.PublicKey()
	if err != nil {
		return fmt.Errorf("signer public key: %w", err)
	}
	ownerBytes, err := crypto.NewEthereumAddress(*pubKey)
	if err != nil {
		return fmt.Errorf("ethereum address: %w", err)
	}
	ownerHex := hex.EncodeToString(ownerBytes)

	// Create postage batch on node0
	batchID0, err := node0.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id: %w", node0.Name(), err)
	}
	c.logger.Infof("socconvergence: created batchID %s on node %s", batchID0, node0.Name())

	// -------------------------------------------------------------------------
	// Scenario 1: Divergent SOC Tie-Break Convergence (SWIP-101)
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 1: Divergent SOC Tie-Break Convergence ---")
	if err := c.testDivergentSOCTieBreak(ctx, fullNodeClients, node0, node1, ownerHex, signer, batchID0, o); err != nil {
		return fmt.Errorf("scenario 1 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 2: Multi-Batch Divergent SOC Convergence
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 2: Multi-Batch Divergent SOC Convergence ---")
	batchID1, err := node1.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel+"-alt")
	if err != nil {
		return fmt.Errorf("node %s: alt batch id: %w", node1.Name(), err)
	}
	if err := c.testMultiBatchDivergence(ctx, fullNodeClients, node0, node1, ownerHex, signer, batchID0, batchID1, o); err != nil {
		return fmt.Errorf("scenario 2 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 3: Identical Payload Checksum Matching
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 3: Identical Payload Checksum Matching ---")
	if err := c.testIdenticalSOCChecksum(ctx, fullNodeClients, node0, node1, ownerHex, signer, batchID0, o); err != nil {
		return fmt.Errorf("scenario 3 failed: %w", err)
	}

	c.logger.Infof("socconvergence: all integration test scenarios passed successfully!")
	return nil
}

func (c *Check) testDivergentSOCTieBreak(
	ctx context.Context,
	fullNodes []*bee.Client,
	node0, node1 *bee.Client,
	ownerHex string,
	signer crypto.Signer,
	batchID string,
	o Options,
) error {
	idBytes, err := randomID()
	if err != nil {
		return err
	}
	idHex := hex.EncodeToString(idBytes)

	// Create 2 divergent payloads wrapping different CACs
	payload1 := []byte("soc-convergence-payload-alpha-101")
	payload2 := []byte("soc-convergence-payload-beta-101")

	ch1, err := cac.New(payload1)
	if err != nil {
		return fmt.Errorf("cac 1: %w", err)
	}
	ch2, err := cac.New(payload2)
	if err != nil {
		return fmt.Errorf("cac 2: %w", err)
	}

	soc1, err := soc.New(idBytes, ch1).Sign(signer)
	if err != nil {
		return fmt.Errorf("sign soc 1: %w", err)
	}
	soc2, err := soc.New(idBytes, ch2).Sign(signer)
	if err != nil {
		return fmt.Errorf("sign soc 2: %w", err)
	}

	// Extract signatures
	sig1Hex := hex.EncodeToString(soc1.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	sig2Hex := hex.EncodeToString(soc2.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])

	socAddress := soc1.Address()
	c.logger.Infof("divergent soc address: %s", socAddress.String())

	// Determine theoretical tie-break winner (lexicographically lower wrapped CAC)
	wins, err := divergentChunkWins(soc1, soc2)
	if err != nil {
		return fmt.Errorf("divergent chunk wins comparison: %w", err)
	}

	var expectedWinnerPayload []byte
	var winnerName string
	if wins {
		expectedWinnerPayload = ch2.Data()
		winnerName = "soc2 (payload2)"
	} else {
		expectedWinnerPayload = ch1.Data()
		winnerName = "soc1 (payload1)"
	}
	c.logger.Infof("theoretical tie-break winner: %s", winnerName)

	// Upload soc1 to node0
	ref1, err := node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload soc1 to node %s: %w", node0.Name(), err)
	}
	c.logger.Infof("uploaded soc1 to node %s: ref %s", node0.Name(), ref1.String())

	// Upload soc2 to node1
	ref2, err := node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload soc2 to node %s: %w", node1.Name(), err)
	}
	c.logger.Infof("uploaded soc2 to node %s: ref %s", node1.Name(), ref2.String())

	// Wait for cluster pullsync convergence across all full nodes
	c.logger.Infof("waiting for pullsync cluster convergence on address %s...", socAddress.String())
	return c.assertClusterConvergence(ctx, fullNodes, socAddress, expectedWinnerPayload, o)
}

func (c *Check) testMultiBatchDivergence(
	ctx context.Context,
	fullNodes []*bee.Client,
	node0, node1 *bee.Client,
	ownerHex string,
	signer crypto.Signer,
	batchID0, batchID1 string,
	o Options,
) error {
	idBytes, err := randomID()
	if err != nil {
		return err
	}
	idHex := hex.EncodeToString(idBytes)

	payload1 := []byte("multi-batch-payload-gamma-201")
	payload2 := []byte("multi-batch-payload-delta-202")

	ch1, err := cac.New(payload1)
	if err != nil {
		return err
	}
	ch2, err := cac.New(payload2)
	if err != nil {
		return err
	}

	soc1, err := soc.New(idBytes, ch1).Sign(signer)
	if err != nil {
		return err
	}
	soc2, err := soc.New(idBytes, ch2).Sign(signer)
	if err != nil {
		return err
	}

	sig1Hex := hex.EncodeToString(soc1.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	sig2Hex := hex.EncodeToString(soc2.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	socAddress := soc1.Address()

	wins, err := divergentChunkWins(soc1, soc2)
	if err != nil {
		return err
	}
	var expectedWinnerPayload []byte
	if wins {
		expectedWinnerPayload = ch2.Data()
	} else {
		expectedWinnerPayload = ch1.Data()
	}

	// Upload to node0 under batchID0
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID0)
	if err != nil {
		return fmt.Errorf("upload soc1 multi-batch: %w", err)
	}

	// Upload to node1 under batchID1
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID1)
	if err != nil {
		return fmt.Errorf("upload soc2 multi-batch: %w", err)
	}

	return c.assertClusterConvergence(ctx, fullNodes, socAddress, expectedWinnerPayload, o)
}

func (c *Check) testIdenticalSOCChecksum(
	ctx context.Context,
	fullNodes []*bee.Client,
	node0, node1 *bee.Client,
	ownerHex string,
	signer crypto.Signer,
	batchID string,
	o Options,
) error {
	idBytes, err := randomID()
	if err != nil {
		return err
	}
	idHex := hex.EncodeToString(idBytes)

	payload := []byte("identical-soc-checksum-payload-301")
	ch, err := cac.New(payload)
	if err != nil {
		return err
	}

	sch, err := soc.New(idBytes, ch).Sign(signer)
	if err != nil {
		return err
	}

	sigHex := hex.EncodeToString(sch.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	socAddress := sch.Address()

	// Upload exact same SOC to node0 and node1
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sigHex, ch.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload identical to node0: %w", err)
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sigHex, ch.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload identical to node1: %w", err)
	}

	return c.assertClusterConvergence(ctx, fullNodes, socAddress, ch.Data(), o)
}

func (c *Check) assertClusterConvergence(
	ctx context.Context,
	nodes []*bee.Client,
	address swarm.Address,
	expectedPayload []byte,
	o Options,
) error {
	ticker := time.NewTicker(o.SyncRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for cluster convergence on %s: %w", address.String(), ctx.Err())
		case <-ticker.C:
			allConverged := true
			for _, node := range nodes {
				retrieved, err := node.DownloadChunk(ctx, address, "", nil)
				if err != nil {
					allConverged = false
					c.logger.Debugf("node %s: download %s error: %v", node.Name(), address.String(), err)
					break
				}
				if !bytes.Contains(retrieved, expectedPayload) && !bytes.Equal(retrieved, expectedPayload) {
					allConverged = false
					c.logger.Debugf("node %s: data mismatch on %s", node.Name(), address.String())
					break
				}
			}

			if allConverged {
				c.logger.Infof("all %d full nodes converged on address %s with expected payload!", len(nodes), address.String())
				return nil
			}
		}
	}
}

func randomID() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// divergentChunkWins reports whether ch2 wins over ch1 by having a
// lexicographically lower wrapped CAC address.
func divergentChunkWins(ch1, ch2 swarm.Chunk) (bool, error) {
	s1, err := soc.FromChunk(ch1)
	if err != nil {
		return false, fmt.Errorf("soc 1 from chunk: %w", err)
	}
	s2, err := soc.FromChunk(ch2)
	if err != nil {
		return false, fmt.Errorf("soc 2 from chunk: %w", err)
	}
	return bytes.Compare(s2.WrappedChunk().Address().Bytes(), s1.WrappedChunk().Address().Bytes()) < 0, nil
}
