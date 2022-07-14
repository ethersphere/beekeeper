package settlements

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	DryRun             bool
	ExpectSettlements  bool
	FileName           string
	FileSize           int64
	GasPrice           string
	PostageAmount      int64
	PostageDepth       uint64
	PostageLabel       string
	Seed               int64
	Threshold          int64 // balances treshold
	UploadNodeCount    int
	WaitBeforeDownload time.Duration // seconds to wait before downloading a file
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		DryRun:             false,
		ExpectSettlements:  true,
		FileName:           "settlements",
		FileSize:           1 * 1024 * 1024, // 1mb
		GasPrice:           "",
		PostageAmount:      1,
		PostageDepth:       16,
		PostageLabel:       "test-label",
		Seed:               0,
		Threshold:          10000000000000,
		UploadNodeCount:    1,
		WaitBeforeDownload: 5 * time.Second,
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		c.logger.Info("running settlements (dry mode)")
		return dryRun(ctx, cluster, o, c.logger)
	}
	c.logger.Info("running settlements")

	rnd := random.PseudoGenerator(o.Seed)
	c.logger.Infof("Seed: %d", o.Seed)

	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	// Initial settlement validation
	balances, err := cluster.FlattenBalances(ctx)
	if err != nil {
		return err
	}
	settlements, err := cluster.FlattenSettlements(ctx)
	if err != nil {
		return err
	}
	if err := validateSettlements(o.Threshold, overlays, balances, settlements, c.logger); err != nil {
		return fmt.Errorf("invalid initial settlements: %s", err.Error())
	}
	c.logger.Info("Settlements are valid")

	var previousSettlements map[string]map[string]orchestration.SentReceived

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.NodeNames()
	for i := 0; i < o.UploadNodeCount; i++ {
		settlementsHappened := false
		// upload file to random node
		uIndex := rnd.Intn(cluster.Size())
		uNode := sortedNodes[uIndex]
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)

		client := clients[uNode]

		c.logger.Info("node", uNode)
		batchID, err := client.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
		if err != nil {
			return fmt.Errorf("node %s: batch id %w", uNode, err)
		}
		c.logger.Infof("node %s: batch id %s", uNode, batchID)

		if err := client.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("node %s: %w", uNode, err)
		}
		c.logger.Infof("File %s uploaded successfully to node %s", file.Address().String(), overlays[uNode].String())

		settlementsValid := false
		// validate settlements after uploading a file
		previousSettlements = settlements
		for t := 0; t < 7; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = cluster.FlattenBalances(ctx)
			if err != nil {
				return err
			}

			settlements, err = cluster.FlattenSettlements(ctx)
			if err != nil {
				return err
			}
			if settlementsHaveHappened(settlements, previousSettlements, c.logger) {
				settlementsHappened = true
			}

			err = validateSettlements(o.Threshold, overlays, balances, settlements, c.logger)
			if err != nil {
				c.logger.Infof("Invalid settlements after uploading a file: %s", err.Error())
				c.logger.Info("Retrying ...")
				continue
			}

			c.logger.Info("Settlements are valid")
			settlementsValid = true
			break
		}

		if !settlementsValid {
			return errors.New("settlements are not valid")
		}

		time.Sleep(o.WaitBeforeDownload)
		// download file from random node
		dIndex := randomIndex(rnd, cluster.Size(), uIndex)
		dNode := sortedNodes[dIndex]
		size, hash, err := clients[dNode].DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %s: %w", dNode, err)
		}
		if !bytes.Equal(file.Hash(), hash) {
			return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.Address().String(), overlays[dNode].String(), file.Size(), size)
		}
		c.logger.Infof("File %s downloaded successfully from node %s", file.Address().String(), overlays[dNode].String())

		settlementsValid = false
		// validate settlements after downloading a file
		previousSettlements = settlements
		for t := 0; t < 7; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = cluster.FlattenBalances(ctx)
			if err != nil {
				return err
			}

			settlements, err = cluster.FlattenSettlements(ctx)
			if err != nil {
				return err
			}

			if settlementsHaveHappened(settlements, previousSettlements, c.logger) {
				settlementsHappened = true
			}

			err = validateSettlements(o.Threshold, overlays, balances, settlements, c.logger)
			if err != nil {
				c.logger.Infof("Invalid settlements after downloading a file: %s", err.Error())
				c.logger.Info("Retrying ...")
				continue
			}

			if !settlementsHappened && o.ExpectSettlements {
				return errors.New("settlements have not happened")
			}

			c.logger.Info("Settlements are valid")
			settlementsValid = true
			break
		}

		if !settlementsValid {
			return errors.New("settlements are not valid")
		}
	}

	return
}

// dryRun executes settlements validation check without files uploading/downloading
func dryRun(ctx context.Context, cluster orchestration.Cluster, o Options, logger logging.Logger) (err error) {
	overlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	balances, err := cluster.FlattenBalances(ctx)
	if err != nil {
		return err
	}

	settlements, err := cluster.FlattenSettlements(ctx)
	if err != nil {
		return err
	}

	if err := validateSettlements(o.Threshold, overlays, balances, settlements, logger); err != nil {
		return fmt.Errorf("invalid settlements")
	}
	logger.Info("Settlements are valid")

	return
}

// validateSettlements checks if settlements are valid
func validateSettlements(threshold int64, overlays orchestration.NodeGroupOverlays, balances orchestration.NodeGroupBalances, settlements orchestration.NodeGroupSettlements, logger logging.Logger) (err error) {
	// threshold validation
	for node, v := range balances {
		for _, balance := range v {
			if balance > threshold {
				return fmt.Errorf("node %s has balance %d that exceeds threshold %d", node, balance, threshold)
			}
		}
	}

	// check balance symmetry
	var noBalanceSymmetry bool
	for node, v := range balances {
		for peer, balance := range v {
			diff := balance + balances[peer][node]
			if diff != 0 {
				logger.Infof("Node %s has asymmetric balance with peer %s", node, peer)
				logger.Infof("Node %s has balance %d with peer %s", node, balance, peer)
				logger.Infof("Peer %s has balance %d with node %s", peer, balances[peer][node], node)
				logger.Infof("Difference: %d", diff)
				noBalanceSymmetry = true
			}
		}
	}
	if noBalanceSymmetry {
		return fmt.Errorf("invalid balances: no symmetry")
	}

	// check settlements symmetry
	var nosettlementsSentymmetry bool
	for node, v := range settlements {
		for peer, settlement := range v {
			diff := settlement.Received - settlements[peer][node].Sent
			if diff != 0 {
				logger.Infof("Node %s has asymmetric settlement with peer %s", node, peer)
				logger.Infof("Node %s received %d from peer %s", node, settlement.Received, peer)
				logger.Infof("Peer %s sent %d to node %s", peer, settlements[peer][node].Sent, node)
				logger.Infof("Difference: %d", diff)
				nosettlementsSentymmetry = true
			}
		}
	}
	if nosettlementsSentymmetry {
		logger.Info("invalid settlements: no symmetry")
	}

	return
}

// settlementsHaveHappened checks if settlements have happened
func settlementsHaveHappened(current, previous map[string]map[string]orchestration.SentReceived, logger logging.Logger) bool {
	for node, v := range current {
		for peer, settlement := range v {
			if settlement.Received != previous[node][peer].Received || settlement.Sent != previous[node][peer].Sent {
				logger.Info("Settlements have happened")
				return true
			}
		}
	}

	return false
}

// randomIndex finds random index <max and not equal to unallowed
func randomIndex(rnd *rand.Rand, max int, unallowed int) (index int) {
	found := false
	for !found {
		index = rnd.Intn(max)
		if index != unallowed {
			found = true
		}
	}

	return
}
