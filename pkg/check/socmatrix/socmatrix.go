// Copyright 2026 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package socmatrix

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
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options holds options for the socmatrix check.
type Options struct {
	GasPrice          string
	PostageTTL        time.Duration
	PostageDepth      uint64
	PostageLabel      string
	RequestTimeout    time.Duration
	SyncRetryInterval time.Duration
}

// NewDefaultOptions returns default options for the check.
func NewDefaultOptions() Options {
	return Options{
		GasPrice:          "",
		PostageTTL:        24 * time.Hour,
		PostageDepth:      16,
		PostageLabel:      "socmatrix-label",
		RequestTimeout:    10 * time.Minute,
		SyncRetryInterval: 2 * time.Second,
	}
}

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
		return fmt.Errorf("socmatrix test requires at least 2 full nodes")
	}

	node0 := fullNodeClients[0]
	node1 := fullNodeClients[1]

	c.logger.Infof("socmatrix: running 10-scenario convergence matrix against %d full nodes (node0: '%s', node1: '%s')",
		len(fullNodeClients), node0.Name(), node1.Name())

	// Shared signer for owner 1
	signer1, ownerHex1, err := newTestSigner()
	if err != nil {
		return err
	}
	// Signer for owner 2 (multi-owner isolation scenario)
	signer2, ownerHex2, err := newTestSigner()
	if err != nil {
		return err
	}

	// Create postage batches on node0 and node1
	batchID0, err := node0.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id: %w", node0.Name(), err)
	}
	batchID1, err := node1.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel+"-alt")
	if err != nil {
		return fmt.Errorf("node %s: alt batch id: %w", node1.Name(), err)
	}

	// -------------------------------------------------------------------------
	// Scenario 1: Divergent SOC Tie-Break (Equal Timestamp, Same Stamp)
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 1: Divergent SOC Tie-Break (Same Stamp) ---")
	if err := c.testDivergentSOCTieBreak(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o); err != nil {
		return fmt.Errorf("scenario 1 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 2: Divergent SOC Timestamp Progression (Increasing Timestamp)
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 2: Timestamp Progression (Increasing Timestamp) ---")
	if err := c.testTimestampProgression(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o); err != nil {
		return fmt.Errorf("scenario 2 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 3: Cross-Batch Equal Timestamp Stamp-Hash Precedence
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 3: Equal Timestamp Cross-Batch Stamp Hash Precedence ---")
	if err := c.testCrossBatchStampHashPrecedence(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, batchID1, o); err != nil {
		return fmt.Errorf("scenario 3 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 4: Multi-Stamp Re-Offer Precedence
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 4: Multi-Stamp Re-Offer Precedence ---")
	if err := c.testMultiStampReofferPrecedence(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, batchID1, o); err != nil {
		return fmt.Errorf("scenario 4 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 5: Multi-Batch Sibling Sum Refresh
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 5: Multi-Batch Sibling Sum Refresh ---")
	if err := c.testMultiBatchSiblingSumRefresh(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, batchID1, o); err != nil {
		return fmt.Errorf("scenario 5 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 6: Identical Payload Checksum Matching (Duplicate De-duplication)
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 6: Identical Payload Checksum Matching ---")
	if err := c.testIdenticalChecksumMatching(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o); err != nil {
		return fmt.Errorf("scenario 6 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 7: CAC vs CAC Stamp Index Collision
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 7: CAC vs CAC Stamp Index Collision ---")
	if err := c.testCACIndexCollision(ctx, fullNodeClients, node0, node1, batchID0, o); err != nil {
		return fmt.Errorf("scenario 7 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 8: CAC vs SOC Stamp Index Tie-Break
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 8: CAC vs SOC Stamp Index Tie-Break ---")
	if err := c.testCACvsSOCTieBreak(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o); err != nil {
		return fmt.Errorf("scenario 8 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 9: 3-Payload Sequenced Divergent SOC Chain
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 9: 3-Payload Sequenced Divergent SOC Chain ---")
	if err := c.testSequencedDivergentChain(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o); err != nil {
		return fmt.Errorf("scenario 9 failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Scenario 10: Multi-Owner Independent SOC Isolation
	// -------------------------------------------------------------------------
	c.logger.Infof("--- Scenario 10: Multi-Owner Independent SOC Isolation ---")
	if err := c.testMultiOwnerSOCIsolation(ctx, fullNodeClients, node0, node1, ownerHex1, ownerHex2, signer1, signer2, batchID0, o); err != nil {
		return fmt.Errorf("scenario 10 failed: %w", err)
	}

	c.logger.Infof("socmatrix: all 10 matrix scenarios completed successfully across %d full nodes!", len(fullNodeClients))
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

	ch1, err := cac.New([]byte("matrix-s1-payload-alpha"))
	if err != nil {
		return err
	}
	ch2, err := cac.New([]byte("matrix-s1-payload-beta"))
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

	wins, err := divergentChunkWins(soc1, soc2)
	if err != nil {
		return err
	}
	var expectedPayload []byte
	if wins {
		expectedPayload = ch2.Data()
	} else {
		expectedPayload = ch1.Data()
	}

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload soc1: %w", err)
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload soc2: %w", err)
	}

	return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
}

func (c *Check) testTimestampProgression(
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

	ch1, err := cac.New([]byte("matrix-s2-initial-payload"))
	if err != nil {
		return err
	}
	ch2, err := cac.New([]byte("matrix-s2-updated-payload"))
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

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload initial soc1: %w", err)
	}

	time.Sleep(1 * time.Second)

	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload updated soc2: %w", err)
	}

	return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), ch2.Data(), o)
}

func (c *Check) testCrossBatchStampHashPrecedence(
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

	ch1, err := cac.New([]byte("matrix-s3-payload-one"))
	if err != nil {
		return err
	}
	ch2, err := cac.New([]byte("matrix-s3-payload-two"))
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

	wins, err := divergentChunkWins(soc1, soc2)
	if err != nil {
		return err
	}
	var expectedPayload []byte
	if wins {
		expectedPayload = ch2.Data()
	} else {
		expectedPayload = ch1.Data()
	}

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID0)
	if err != nil {
		return err
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID1)
	if err != nil {
		return err
	}

	return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
}

func (c *Check) testMultiStampReofferPrecedence(
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

	ch1, err := cac.New([]byte("matrix-s4-payload-alpha"))
	if err != nil {
		return err
	}
	ch2, err := cac.New([]byte("matrix-s4-payload-beta"))
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

	wins, err := divergentChunkWins(soc1, soc2)
	if err != nil {
		return err
	}
	var expectedPayload []byte
	if wins {
		expectedPayload = ch2.Data()
	} else {
		expectedPayload = ch1.Data()
	}

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID0)
	if err != nil {
		return err
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID1)
	if err != nil {
		return err
	}

	if err := c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o); err != nil {
		return err
	}

	// Re-offer soc1 under batchID0
	_, _ = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID0)
	return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
}

func (c *Check) testMultiBatchSiblingSumRefresh(
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

	ch1, err := cac.New([]byte("matrix-s5-initial-multibatch"))
	if err != nil {
		return err
	}
	ch2, err := cac.New([]byte("matrix-s5-updated-multibatch"))
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

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID0)
	if err != nil {
		return err
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID1)
	if err != nil {
		return err
	}

	if err := c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), ch1.Data(), o); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID0)
	if err != nil {
		return err
	}

	return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), ch2.Data(), o)
}

func (c *Check) testIdenticalChecksumMatching(
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

	ch, err := cac.New([]byte("matrix-s6-identical-payload"))
	if err != nil {
		return err
	}
	sch, err := soc.New(idBytes, ch).Sign(signer)
	if err != nil {
		return err
	}

	sigHex := hex.EncodeToString(sch.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sigHex, ch.Data(), batchID)
	if err != nil {
		return err
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sigHex, ch.Data(), batchID)
	if err != nil {
		return err
	}

	return c.assertClusterConvergence(ctx, fullNodes, sch.Address(), ch.Data(), o)
}

func (c *Check) testCACIndexCollision(
	ctx context.Context,
	fullNodes []*bee.Client,
	node0, node1 *bee.Client,
	batchID string,
	o Options,
) error {
	ch1, err := cac.New([]byte("matrix-s7-cac-payload-one"))
	if err != nil {
		return err
	}
	ch2, err := cac.New([]byte("matrix-s7-cac-payload-two"))
	if err != nil {
		return err
	}

	_, err = node0.UploadBytes(ctx, ch1.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("upload cac1: %w", err)
	}
	_, err = node1.UploadBytes(ctx, ch2.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("upload cac2: %w", err)
	}

	if err := c.assertClusterConvergence(ctx, fullNodes, ch1.Address(), ch1.Data(), o); err != nil {
		return err
	}
	return c.assertClusterConvergence(ctx, fullNodes, ch2.Address(), ch2.Data(), o)
}

func (c *Check) testCACvsSOCTieBreak(
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

	chCAC, err := cac.New([]byte("matrix-s8-cac-content"))
	if err != nil {
		return err
	}
	socChunk, err := soc.New(idBytes, chCAC).Sign(signer)
	if err != nil {
		return err
	}

	sigHex := hex.EncodeToString(socChunk.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])

	_, err = node0.UploadBytes(ctx, chCAC.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	_, err = node1.UploadSOC(ctx, ownerHex, idHex, sigHex, chCAC.Data(), batchID)
	if err != nil {
		return err
	}

	if err := c.assertClusterConvergence(ctx, fullNodes, chCAC.Address(), chCAC.Data(), o); err != nil {
		return err
	}
	return c.assertClusterConvergence(ctx, fullNodes, socChunk.Address(), chCAC.Data(), o)
}

func (c *Check) testSequencedDivergentChain(
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

	p1 := []byte("matrix-s9-chain-payload-one")
	p2 := []byte("matrix-s9-chain-payload-two")
	p3 := []byte("matrix-s9-chain-payload-three")

	ch1, _ := cac.New(p1)
	ch2, _ := cac.New(p2)
	ch3, _ := cac.New(p3)

	soc1, _ := soc.New(idBytes, ch1).Sign(signer)
	soc2, _ := soc.New(idBytes, ch2).Sign(signer)
	soc3, _ := soc.New(idBytes, ch3).Sign(signer)

	sig1Hex := hex.EncodeToString(soc1.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	sig2Hex := hex.EncodeToString(soc2.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	sig3Hex := hex.EncodeToString(soc3.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])

	_, _ = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID)
	_, _ = node1.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
	_, _ = node0.UploadSOC(ctx, ownerHex, idHex, sig3Hex, ch3.Data(), batchID)

	// Determine winner among all three
	win12, _ := divergentChunkWins(soc1, soc2)
	var winner12 swarm.Chunk
	if win12 {
		winner12 = soc2
	} else {
		winner12 = soc1
	}
	winFinal, _ := divergentChunkWins(winner12, soc3)
	var winningSOC swarm.Chunk
	if winFinal {
		winningSOC = soc3
	} else {
		winningSOC = winner12
	}

	var expectedPayload []byte
	if winningSOC.Address().Equal(soc1.Address()) && bytes.Equal(winningSOC.Data(), soc1.Data()) {
		expectedPayload = ch1.Data()
	} else if winningSOC.Address().Equal(soc2.Address()) && bytes.Equal(winningSOC.Data(), soc2.Data()) {
		expectedPayload = ch2.Data()
	} else {
		expectedPayload = ch3.Data()
	}

	return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
}

func (c *Check) testMultiOwnerSOCIsolation(
	ctx context.Context,
	fullNodes []*bee.Client,
	node0, node1 *bee.Client,
	ownerHex1, ownerHex2 string,
	signer1, signer2 crypto.Signer,
	batchID string,
	o Options,
) error {
	idBytes, err := randomID()
	if err != nil {
		return err
	}
	idHex := hex.EncodeToString(idBytes)

	ch1, _ := cac.New([]byte("matrix-s10-owner1-payload"))
	ch2, _ := cac.New([]byte("matrix-s10-owner2-payload"))

	soc1, _ := soc.New(idBytes, ch1).Sign(signer1)
	soc2, _ := soc.New(idBytes, ch2).Sign(signer2)

	sig1Hex := hex.EncodeToString(soc1.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])
	sig2Hex := hex.EncodeToString(soc2.Data()[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize])

	_, err = node0.UploadSOC(ctx, ownerHex1, idHex, sig1Hex, ch1.Data(), batchID)
	if err != nil {
		return err
	}
	_, err = node1.UploadSOC(ctx, ownerHex2, idHex, sig2Hex, ch2.Data(), batchID)
	if err != nil {
		return err
	}

	if err := c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), ch1.Data(), o); err != nil {
		return err
	}
	return c.assertClusterConvergence(ctx, fullNodes, soc2.Address(), ch2.Data(), o)
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

func newTestSigner() (crypto.Signer, string, error) {
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return nil, "", err
	}
	signer := crypto.NewDefaultSigner(privKey)
	pubKey, err := signer.PublicKey()
	if err != nil {
		return nil, "", err
	}
	ownerBytes, err := crypto.NewEthereumAddress(*pubKey)
	if err != nil {
		return nil, "", err
	}
	return signer, hex.EncodeToString(ownerBytes), nil
}

func randomID() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

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
