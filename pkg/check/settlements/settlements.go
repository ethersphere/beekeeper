package settlements

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents settlements check options
type Options struct {
	UploadNodeCount int
	FileName        string
	FileSize        int64
	Seed            int64
	Threshold       int
}

// Check executes settlements check
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < o.UploadNodeCount; i++ {
		fmt.Printf("Validate settlements before uploading a file:\n")
		b, err := c.Balances(ctx)
		if err != nil {
			return err
		}
		s, err := c.Settlements(ctx)
		if err != nil {
			return err
		}
		if err := validateSettlements(o.Threshold, overlays, b, s); err != nil {
			return fmt.Errorf("invalid settlements before uploading a file: %s", err.Error())
		}
		fmt.Printf("Valid settlements\n")

		// upload file to random node
		uIndex := rnd.Intn(c.Size())
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
		if err := c.Nodes[uIndex].UploadFile(ctx, &file); err != nil {
			return fmt.Errorf("node %d: %w", uIndex, err)
		}
		fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[uIndex].String())

		fmt.Printf("Validate settlements after uploading a file:\n")
		b, err = c.Balances(ctx)
		if err != nil {
			return err
		}
		s, err = c.Settlements(ctx)
		if err != nil {
			return err
		}
		if err := validateSettlements(o.Threshold, overlays, b, s); err != nil {
			return fmt.Errorf("invalid settlements after uploading a file: %s", err.Error())
		}
		fmt.Printf("Valid settlements\n")

		// download file from random node
		dIndex := randomIndex(rnd, c.Size(), uIndex)
		size, hash, err := c.Nodes[dIndex].DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %d: %w", dIndex, err)
		}
		if !bytes.Equal(file.Hash(), hash) {
			return fmt.Errorf("File %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.Address().String(), overlays[dIndex].String(), file.Size(), size)
		}
		fmt.Printf("File downloaded successfully %s from node %s\n", file.Address().String(), overlays[dIndex].String())

		fmt.Printf("Validate settlements after downloading a file:\n")
		b, err = c.Balances(ctx)
		if err != nil {
			return err
		}
		s, err = c.Settlements(ctx)
		if err != nil {
			return err
		}
		if err := validateSettlements(o.Threshold, overlays, b, s); err != nil {
			return fmt.Errorf("invalid settlements after downloading a file: %s", err.Error())
		}
		fmt.Printf("Valid settlements\n")
	}

	return
}

// DryRunCheck executes settlements validation check without files uploading/downloading
func DryRunCheck(c bee.Cluster, o Options) (err error) {
	ctx := context.Background()

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	balances, err := c.Balances(ctx)
	if err != nil {
		return err
	}

	settlements, err := c.Settlements(ctx)
	if err != nil {
		return err
	}

	if err := validateSettlements(o.Threshold, overlays, balances, settlements); err != nil {
		return fmt.Errorf("invalid settlements")
	}
	fmt.Printf("Valid settlements\n")

	return
}

// validateSettlements checks if settlements are valid
func validateSettlements(threshold int, overlays []swarm.Address, balances map[string]map[string]int, settlements map[string]map[string]bee.SentReceived) (err error) {
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
				fmt.Printf("Node %s has asymmetric balance with peer %s\n", node, peer)
				fmt.Printf("Node %s has balance %d with peer %s\n", node, balance, peer)
				fmt.Printf("Peer %s has balance %d with node %s\n", peer, balances[peer][node], node)
				fmt.Printf("Difference: %d\n", diff)
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
				fmt.Printf("Node %s has asymmetric settlement with peer %s\n", node, peer)
				fmt.Printf("Node %s received %d from peer %s\n", node, settlement.Received, peer)
				fmt.Printf("Peer %s sent %d to node %s\n", peer, settlements[peer][node].Sent, node)
				fmt.Printf("Difference: %d\n", diff)
				nosettlementsSentymmetry = true
			}
		}
	}
	if nosettlementsSentymmetry {
		fmt.Printf("invalid settlements: no symmetry\n")
	}

	return
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
