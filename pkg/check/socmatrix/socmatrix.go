// Copyright 2026 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package socmatrix

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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

// Options holds options for the socmatrix check.
type Options struct {
	PostageTTL        time.Duration
	PostageDepth      uint64
	PostageLabel      string
	RequestTimeout    time.Duration
	SyncRetryInterval time.Duration
	// SyncWait is used by precision (former chunk-convergence) scenarios.
	SyncWait time.Duration
	// Password decrypts the batch-issuer Swarm key for precision stamp resigning.
	Password string
}

// NewDefaultOptions returns default options for the check.
func NewDefaultOptions() Options {
	return Options{
		PostageTTL:        24 * time.Hour,
		PostageDepth:      16,
		PostageLabel:      "socmatrix-label",
		RequestTimeout:    10 * time.Minute,
		SyncRetryInterval: 2 * time.Second,
		SyncWait:          45 * time.Second,
		Password:          "beekeeper",
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

	c.logger.Infof("socmatrix: running API-stamp matrix (scenarios 1-10) against %d full nodes (node0: '%s', node1: '%s')",
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

	// Try setting up batch issuer and signer for precomputed equal-timestamp stamp tests
	issuer, batchSigner, _, _ := c.setupIssuerBatch(ctx, cluster, o)

	// API-stamp matrix (1-10): continue-on-error so later scenarios still run.
	var matrixErrs error
	runMatrix := func(name string, fn func() error) {
		c.logger.Infof("--- %s ---", name)
		if err := fn(); err != nil {
			c.logger.Errorf("%s failed: %v", name, err)
			matrixErrs = errors.Join(matrixErrs, fmt.Errorf("%s: %w", name, err))
			return
		}
		c.logger.Infof("%s passed", name)
	}

	runMatrix("Scenario 1: Divergent SOC Tie-Break (Same Stamp)", func() error {
		return c.testDivergentSOCTieBreak(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, issuer, batchSigner, o)
	})
	runMatrix("Scenario 2: Timestamp Progression (Increasing Timestamp)", func() error {
		return c.testTimestampProgression(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o)
	})
	runMatrix("Scenario 3: Equal Timestamp Cross-Batch Stamp Hash Precedence", func() error {
		return c.testCrossBatchStampHashPrecedence(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, batchID1, issuer, batchSigner, o)
	})
	runMatrix("Scenario 4: Multi-Stamp Re-Offer Precedence", func() error {
		return c.testMultiStampReofferPrecedence(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, batchID1, issuer, batchSigner, o)
	})
	runMatrix("Scenario 5: Multi-Batch Sibling Sum Refresh", func() error {
		return c.testMultiBatchSiblingSumRefresh(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, batchID1, o)
	})
	runMatrix("Scenario 6: Identical Payload Checksum Matching", func() error {
		return c.testIdenticalChecksumMatching(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o)
	})
	runMatrix("Scenario 7: CAC vs CAC Stamp Index Collision", func() error {
		return c.testCACIndexCollision(ctx, fullNodeClients, node0, node1, batchID0, o)
	})
	runMatrix("Scenario 8: CAC vs SOC Stamp Index Tie-Break", func() error {
		return c.testCACvsSOCTieBreak(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, o)
	})
	runMatrix("Scenario 9: 3-Payload Sequenced Divergent SOC Chain", func() error {
		return c.testSequencedDivergentChain(ctx, fullNodeClients, node0, node1, ownerHex1, signer1, batchID0, issuer, batchSigner, o)
	})
	runMatrix("Scenario 10: Multi-Owner Independent SOC Isolation", func() error {
		return c.testMultiOwnerSOCIsolation(ctx, fullNodeClients, node0, node1, ownerHex1, ownerHex2, signer1, signer2, batchID0, o)
	})

	if matrixErrs != nil {
		c.logger.Errorf("socmatrix: scenarios 1-10 finished with errors across %d full nodes: %v", len(fullNodeClients), matrixErrs)
	} else {
		c.logger.Infof("socmatrix: scenarios 1-10 completed successfully across %d full nodes", len(fullNodeClients))
	}

	n1, n2, err := pickClosestPair(ctx, fullNodeClients)
	if err != nil {
		return err
	}
	c.logger.Infof("precision neighborhood upload pair: %s and %s", n1.Name(), n2.Name())
	neighborhood := []*bee.Client{n1, n2}

	issuer, batchSigner, batchID, err := c.setupIssuerBatch(ctx, cluster, o)
	if err != nil {
		return err
	}
	c.logger.Infof("precision batch issuer=%s batch_id=%s", issuer.Name(), batchID)

	precision := []struct {
		name string
		run  func() error
	}{
		{"11 precision divergent soc identical stamp", func() error {
			return c.scenarioDivergentSOC(ctx, issuer, neighborhood, batchID, o.SyncWait)
		}},
		{"12 precision soc same index higher timestamp", func() error {
			return c.scenarioSOCSameIndexTimestamp(ctx, issuer, batchSigner, neighborhood, batchID, o.SyncWait)
		}},
		{"13 precision soc different index higher timestamp", func() error {
			return c.scenarioSOCDifferentIndexTimestamp(ctx, issuer, neighborhood, batchID, o.SyncWait)
		}},
		{"14 precision cac same index higher timestamp", func() error {
			return c.scenarioCACSameIndexTimestamp(ctx, issuer, batchSigner, neighborhood, batchID, o.SyncWait)
		}},
		{"15 precision cac same index equal timestamp", func() error {
			return c.scenarioCACSameIndexEqualTimestamp(ctx, issuer, batchSigner, neighborhood, batchID, o.SyncWait)
		}},
		{"16 precision soc stale stamp redelivery race", func() error {
			return c.scenarioSOCStaleStampRedeliveryRace(ctx, issuer, batchSigner, neighborhood, batchID, o, o.SyncWait)
		}},
	}

	var errs error
	for _, s := range precision {
		c.logger.Infof("--- Scenario %s ---", s.name)
		if err := s.run(); err != nil {
			c.logger.Errorf("%s failed: %v", s.name, err)
			errs = errors.Join(errs, fmt.Errorf("%s: %w", s.name, err))
			continue
		}
		c.logger.Infof("%s passed", s.name)
	}
	errs = errors.Join(matrixErrs, errs)
	if errs != nil {
		return errs
	}

	c.logger.Infof("socmatrix: all 16 scenarios completed successfully")
	return nil
}

func (c *Check) testDivergentSOCTieBreak(
	ctx context.Context,
	fullNodes []*bee.Client,
	node0, node1 *bee.Client,
	ownerHex string,
	signer crypto.Signer,
	batchID string,
	issuer *bee.Client,
	batchSigner crypto.Signer,
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

	if issuer != nil && batchSigner != nil {
		env, err := issuer.CreateEnvelope(ctx, soc1.Address(), batchID)
		if err == nil {
			stampBytes, err := env.MarshalBinary()
			if err == nil {
				_, err = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), stampBytes)
				if err != nil {
					return fmt.Errorf("upload soc1 with stamp: %w", err)
				}
				_, _ = node1.UploadSOCWithStamp(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), stampBytes)
				return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
			}
		}
	}

	// Fallback to API stamp upload if issuer/batchSigner not configured
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID)
	if err != nil {
		return fmt.Errorf("upload soc1: %w", err)
	}
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
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

	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
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
	issuer *bee.Client,
	batchSigner crypto.Signer,
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

	if issuer != nil && batchSigner != nil {
		batch0Bytes, err0 := hex.DecodeString(batchID0)
		batch1Bytes, err1 := hex.DecodeString(batchID1)
		env0, err2 := issuer.CreateEnvelope(ctx, soc1.Address(), batchID0)
		env1, err3 := issuer.CreateEnvelope(ctx, soc1.Address(), batchID1)
		if err0 == nil && err1 == nil && err2 == nil && err3 == nil {
			ts := binary.BigEndian.Uint64(env0.Timestamp())
			stamp0, errS0 := signStamp(batchSigner, batch0Bytes, env0.Index(), uint64ToBytes(ts), soc1.Address())
			stamp1, errS1 := signStamp(batchSigner, batch1Bytes, env1.Index(), uint64ToBytes(ts), soc1.Address())
			if errS0 == nil && errS1 == nil {
				hash0, errH0 := stamp0.Hash()
				hash1, errH1 := stamp1.Hash()
				if errH0 == nil && errH1 == nil {
					var expectedPayload []byte
					if bytes.Compare(hash1, hash0) < 0 {
						expectedPayload = ch2.Data()
					} else {
						expectedPayload = ch1.Data()
					}

					stamp0Bytes, _ := stamp0.MarshalBinary()
					stamp1Bytes, _ := stamp1.MarshalBinary()

					_, err = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), stamp0Bytes)
					if err != nil {
						return fmt.Errorf("upload soc1 with stamp0: %w", err)
					}
					_, _ = node1.UploadSOCWithStamp(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), stamp1Bytes)
					return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
				}
			}
		}
	}

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
	issuer *bee.Client,
	batchSigner crypto.Signer,
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

	if issuer != nil && batchSigner != nil {
		batch0Bytes, err0 := hex.DecodeString(batchID0)
		batch1Bytes, err1 := hex.DecodeString(batchID1)
		env0, err2 := issuer.CreateEnvelope(ctx, soc1.Address(), batchID0)
		env1, err3 := issuer.CreateEnvelope(ctx, soc1.Address(), batchID1)
		if err0 == nil && err1 == nil && err2 == nil && err3 == nil {
			ts := binary.BigEndian.Uint64(env0.Timestamp())
			stamp0, errS0 := signStamp(batchSigner, batch0Bytes, env0.Index(), uint64ToBytes(ts), soc1.Address())
			stamp1, errS1 := signStamp(batchSigner, batch1Bytes, env1.Index(), uint64ToBytes(ts), soc1.Address())
			if errS0 == nil && errS1 == nil {
				hash0, errH0 := stamp0.Hash()
				hash1, errH1 := stamp1.Hash()
				if errH0 == nil && errH1 == nil {
					var expectedPayload []byte
					if bytes.Compare(hash1, hash0) < 0 {
						expectedPayload = ch2.Data()
					} else {
						expectedPayload = ch1.Data()
					}

					stamp0Bytes, _ := stamp0.MarshalBinary()
					stamp1Bytes, _ := stamp1.MarshalBinary()

					_, err = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), stamp0Bytes)
					if err != nil {
						return fmt.Errorf("upload soc1 with stamp0: %w", err)
					}
					_, _ = node1.UploadSOCWithStamp(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), stamp1Bytes)

					if err := c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o); err != nil {
						return err
					}

					// Re-offer soc1 under batchID0 with stamp0Bytes (same stamp timestamp)
					_, _ = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), stamp0Bytes)
					return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
				}
			}
		}
	}

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
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sigHex, ch.Data(), batchID)
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

	_, err = node0.UploadChunk(ctx, ch1.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("upload cac1: %w", err)
	}
	_, err = node0.UploadChunk(ctx, ch2.Data(), api.UploadOptions{BatchID: batchID})
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

	_, err = node0.UploadChunk(ctx, chCAC.Data(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	_, err = node0.UploadSOC(ctx, ownerHex, idHex, sigHex, chCAC.Data(), batchID)
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
	issuer *bee.Client,
	batchSigner crypto.Signer,
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

	if issuer != nil {
		env, err := issuer.CreateEnvelope(ctx, soc1.Address(), batchID)
		if err == nil {
			stampBytes, err := env.MarshalBinary()
			if err == nil {
				_, _ = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), stampBytes)
				_, _ = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), stampBytes)
				_, _ = node0.UploadSOCWithStamp(ctx, ownerHex, idHex, sig3Hex, ch3.Data(), stampBytes)
				return c.assertClusterConvergence(ctx, fullNodes, soc1.Address(), expectedPayload, o)
			}
		}
	}

	_, _ = node0.UploadSOC(ctx, ownerHex, idHex, sig1Hex, ch1.Data(), batchID)
	_, _ = node0.UploadSOC(ctx, ownerHex, idHex, sig2Hex, ch2.Data(), batchID)
	_, _ = node0.UploadSOC(ctx, ownerHex, idHex, sig3Hex, ch3.Data(), batchID)

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
	_, err = node0.UploadSOC(ctx, ownerHex2, idHex, sig2Hex, ch2.Data(), batchID)
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

	// Bound wait by sync budget, not the parent check timeout (often 15–45m).
	waitFor := 5 * o.SyncWait
	if waitFor < 2*time.Minute {
		waitFor = 2 * time.Minute
	}
	deadline := time.Now().Add(waitFor)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for cluster convergence on %s: %w", address.String(), ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for cluster to converge on %s", address.String())
			}
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
	_, err := crand.Read(key)
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

// ---------------------------------------------------------------------------
// Precision scenarios (former chunk-convergence) and helpers
// ---------------------------------------------------------------------------

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
	// Identical stamp: losing upload may fail the wrapped-CAC tie-break.
	// Require only the first (winning) upload to succeed.
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
	// Parallel uploads: loser (higher address) and winner (lower address).
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
//	SOC₁ under stamp S (batch A) → uploaded to Sam
//	SOC₂ under stamp T (batch B) → uploaded to Sara
//	SOC₃ reuses stamp S with payload P₃ whose wrapped CAC beats P₁ and P₂ →
//	uploaded to both nodes in parallel with the SOC₁/SOC₂ uploads
//
// After sync, every neighborhood node must hold the same payload among P₁/P₂/P₃
// (which one wins does not matter; agreement does).
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
	c.logger.Infof("race uploads finished; waiting %s for neighborhood agreement on any of P1/P2/P3", syncWait)

	time.Sleep(syncWait)

	candidates := []struct {
		label string
		data  []byte
	}{
		{"P1", soc1.Data()},
		{"P2", soc2.Data()},
		{"P3", soc3.Data()},
	}

	agreed, err := assertNeighborhoodAgreesAmong(ctx, c.logger, neighborhood, addr, candidates)
	if err != nil {
		return fmt.Errorf("after stale-stamp race (want same payload on both): %w", err)
	}
	c.logger.Infof("neighborhood converged on %s", agreed)
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

	for attempt := 0; attempt < 5; attempt++ {
		if _, err := n.UploadSOCWithStamp(ctx, owner, idHex, sig, wrapped, stamp); err != nil {
			if strings.Contains(err.Error(), "503") || strings.Contains(err.Error(), "Service Unavailable") {
				time.Sleep(1 * time.Second)
				continue
			}
			return fmt.Errorf("upload to %s: %w", n.Name(), err)
		}
		return nil
	}
	return fmt.Errorf("upload to %s: exceeded 5 retries on 503", n.Name())
}

// uploadSOCPair uploads ch1 to n1 and ch2 to n2 in parallel.
// If requireSecond is false, a failed second upload is ignored (tie-break loser).
func uploadSOCPair(ctx context.Context, n1, n2 *bee.Client, owner string, id []byte, ch1, ch2 swarm.Chunk, stamp1, stamp2 []byte, requireSecond bool) error {
	type result struct {
		which int // 1 or 2
		err   error
	}
	results := make(chan result, 2)

	go func() {
		results <- result{1, uploadSOC(ctx, n1, owner, id, ch1, stamp1)}
	}()
	go func() {
		results <- result{2, uploadSOC(ctx, n2, owner, id, ch2, stamp2)}
	}()

	var err1, err2 error
	for range 2 {
		r := <-results
		switch r.which {
		case 1:
			err1 = r.err
		case 2:
			err2 = r.err
		}
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil && requireSecond {
		return err2
	}
	return nil
}

// uploadChunkPair uploads ch1 to n1 and ch2 to n2 in parallel.
func uploadChunkPair(ctx context.Context, n1, n2 *bee.Client, ch1, ch2 swarm.Chunk) error {
	st1, err := ch1.Stamp().MarshalBinary()
	if err != nil {
		return err
	}
	st2, err := ch2.Stamp().MarshalBinary()
	if err != nil {
		return err
	}

	type result struct {
		name string
		err  error
	}
	results := make(chan result, 2)

	go func() {
		_, err := n1.UploadChunk(ctx, ch1.Data(), api.UploadOptions{Stamp: hex.EncodeToString(st1), Direct: true})
		if err != nil {
			err = fmt.Errorf("upload to %s: %w", n1.Name(), err)
		}
		results <- result{n1.Name(), err}
	}()
	go func() {
		_, err := n2.UploadChunk(ctx, ch2.Data(), api.UploadOptions{Stamp: hex.EncodeToString(st2), Direct: true})
		if err != nil {
			err = fmt.Errorf("upload to %s: %w", n2.Name(), err)
		}
		results <- result{n2.Name(), err}
	}()

	var errs error
	for range 2 {
		r := <-results
		if r.err != nil {
			errs = errors.Join(errs, r.err)
		}
	}
	return errs
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

// assertNeighborhoodAgreesAmong checks that every node holds the same body and
// that body matches one of the labeled candidates. Returns the winning label.
func assertNeighborhoodAgreesAmong(
	ctx context.Context,
	logger logging.Logger,
	nodes []*bee.Client,
	addr swarm.Address,
	candidates []struct {
		label string
		data  []byte
	},
) (string, error) {
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes")
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no candidates")
	}

	matchLabel := func(got []byte) string {
		for _, c := range candidates {
			if bytes.Equal(got, c.data) {
				return c.label
			}
		}
		return ""
	}

	var agreed []byte
	var agreedLabel string
	for _, n := range nodes {
		got, err := n.DownloadChunk(ctx, addr, "", nil)
		if err != nil {
			return "", fmt.Errorf("node %s download %s: %w", n.Name(), addr, err)
		}
		label := matchLabel(got)
		if label == "" {
			logger.Errorf("node %s holds unrecognized payload len=%d", n.Name(), len(got))
			return "", fmt.Errorf("node %s: payload not among candidates", n.Name())
		}
		logger.Infof("%s holds %s", n.Name(), label)
		if agreed == nil {
			agreed = got
			agreedLabel = label
			continue
		}
		if !bytes.Equal(got, agreed) {
			logger.Errorf("node %s holds %s, others hold %s", n.Name(), label, agreedLabel)
			return "", fmt.Errorf("node %s: desync (%s vs %s)", n.Name(), label, agreedLabel)
		}
	}
	return agreedLabel, nil
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
