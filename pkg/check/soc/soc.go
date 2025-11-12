package soc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/replicas/combinator"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
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
	UploadRLevel   redundancy.Level
	DownloadRLevel redundancy.Level
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GasPrice:       "",
		PostageTTL:     24 * time.Hour,
		PostageDepth:   16,
		PostageLabel:   "test-label",
		RequestTimeout: 5 * time.Minute,
		UploadRLevel:   redundancy.PARANOID,
		DownloadRLevel: redundancy.PARANOID,
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

	socOptions := &api.SOCOptions{
		RLevel: &o.UploadRLevel,
	}

	ref, err := node.UploadSOC(ctx, owner, id, sig, ch.Data(), batchID, socOptions)
	if err != nil {
		return fmt.Errorf("node %s: upload soc chunk %w", nodeName, err)
	}

	c.logger.Infof("soc: chunk uploaded to node %s, reference %s", nodeName, ref.String())

	retrieved, err := node.DownloadChunk(ctx, ref, "", nil)
	if err != nil {
		return fmt.Errorf("node %s: download soc chunk %w", nodeName, err)
	}

	c.logger.Infof("soc: original chunk retrieved from node %s", nodeName)

	if !bytes.Equal(retrieved, chunkData) {
		return errors.New("soc: retrieved chunk data does NOT match soc chunk")
	}

	replicaErrs := c.testReplicaRetrieval(ctx, node, nodeName, ref, chunkData, o)

	return replicaErrs
}

// testReplicaRetrieval tests replica retrieval with configurable redundancy levels and automatic error handling
func (c *Check) testReplicaRetrieval(ctx context.Context, node *bee.Client, nodeName string, ref swarm.Address, originalChunkData []byte, opts Options) error {
	countTotal, countGood, countBad := 0, 0, 0
	getFromCache := true

	replicaIter := combinator.IterateReplicaAddresses(ref, int(opts.DownloadRLevel))
	var replicaErrors []error

	uploadExpected := opts.UploadRLevel.GetReplicaCount()
	downloadExpected := opts.DownloadRLevel.GetReplicaCount()

	expectedSuccesses := min(uploadExpected, downloadExpected)
	expectedFailures := downloadExpected - expectedSuccesses

	c.logger.Infof("soc: testing replica retrieval with upload level %s (%d replicas) and download level %s (%d replicas to check)",
		redundancyLevelName(opts.UploadRLevel), uploadExpected,
		redundancyLevelName(opts.DownloadRLevel), downloadExpected)
	c.logger.Infof("soc: expecting %d successful retrievals and %d failures", expectedSuccesses, expectedFailures)

	for addr := range replicaIter {
		countTotal++
		if addr.Equal(ref) {
			c.logger.Errorf("found original chunk address among replicas on position %d", countTotal)
			continue
		}

		retrievedReplica, err := node.DownloadChunk(ctx, addr, "", &api.DownloadOptions{
			Cache: &getFromCache,
		})
		if err != nil {
			countBad++
			c.logger.Infof("node %s: download soc replica chunk %d failed: %v (address: %s)",
				nodeName, countTotal, err, addr.String())
			replicaErrors = append(replicaErrors, err)
		} else {
			countGood++
			c.logger.Infof("soc: replica chunk %d (%s) retrieved successfully from node %s",
				countTotal, addr.String(), nodeName)

			if !bytes.Equal(retrievedReplica, originalChunkData) {
				return fmt.Errorf("soc: retrieved replica chunk %d data does NOT match original soc chunk", countTotal)
			}
		}
	}

	c.logger.Infof("soc: replica retrieval summary for node %s: total=%d, successful=%d, failed=%d",
		nodeName, countTotal, countGood, countBad)

	// Validate total attempts
	if countTotal != downloadExpected {
		return fmt.Errorf("soc: expected to check %d replicas (download level %s), but checked %d",
			downloadExpected, redundancyLevelName(opts.DownloadRLevel), countTotal)
	}

	// Validate expected vs actual results
	if countGood != expectedSuccesses {
		return fmt.Errorf("soc: expected %d successful retrievals but got %d (upload level %s provides %d replicas)",
			expectedSuccesses, countGood, redundancyLevelName(opts.UploadRLevel), uploadExpected)
	}

	if countBad != expectedFailures {
		return fmt.Errorf("soc: expected %d failed retrievals but got %d (difference between %d download attempts and %d available replicas)",
			expectedFailures, countBad, downloadExpected, uploadExpected)
	}

	// Validate that replica errors are the expected type when we have failures
	if expectedFailures > 0 {
		if err := validateReplicaErrors(replicaErrors, expectedFailures); err != nil {
			return fmt.Errorf("soc: replica error validation failed: %w", err)
		}
		c.logger.Infof("soc: validated %d replica errors are expected type (HTTP 500 'read chunk failed')", expectedFailures)
	}

	// Log success
	if expectedFailures > 0 {
		c.logger.Infof("soc: test passed with expected pattern: %d/%d successful retrievals (upload level %s < download level %s)",
			countGood, countTotal, redundancyLevelName(opts.UploadRLevel), redundancyLevelName(opts.DownloadRLevel))
	} else {
		c.logger.Infof("soc: test passed with all %d replicas retrieved successfully (upload level %s >= download level %s)",
			countGood, redundancyLevelName(opts.UploadRLevel), redundancyLevelName(opts.DownloadRLevel))
	}

	return nil
}

// isExpectedReplicaError checks if an error is the expected "read chunk failed" HTTP 500 error
func isExpectedReplicaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for the pattern: response message "read chunk failed": status: 500 Internal Server Error
	return strings.Contains(errStr, `response message "read chunk failed"`) &&
		strings.Contains(errStr, "500 Internal Server Error")
}

// validateReplicaErrors checks that all replica errors are the expected type
func validateReplicaErrors(errors []error, expectedCount int) error {
	if len(errors) != expectedCount {
		return fmt.Errorf("expected %d replica errors, got %d", expectedCount, len(errors))
	}

	for i, err := range errors {
		if !isExpectedReplicaError(err) {
			return fmt.Errorf("replica error %d is not expected type: %w", i+1, err)
		}
	}
	return nil
}

// redundancyLevelName returns a human-readable name for redundancy level
func redundancyLevelName(level redundancy.Level) string {
	switch level {
	case redundancy.NONE:
		return "NONE"
	case redundancy.MEDIUM:
		return "MEDIUM"
	case redundancy.STRONG:
		return "STRONG"
	case redundancy.INSANE:
		return "INSANE"
	case redundancy.PARANOID:
		return "PARANOID"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", level)
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
