package settlements

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents settlements check options
type Options struct {
	NodeGroup          string
	UploadNodeCount    int
	FileName           string
	FileSize           int64
	Seed               int64
	Threshold          int64
	WaitBeforeDownload int
}

// Check executes settlements check
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	// Initial settlement validation
	balances, err := ng.Balances(ctx)
	if err != nil {
		return err
	}
	settlements, err := ng.Settlements(ctx)
	if err != nil {
		return err
	}
	if err := validateSettlements(o.Threshold, overlays, balances, settlements); err != nil {
		return fmt.Errorf("invalid initial settlements: %s", err.Error())
	}
	fmt.Println("Settlements are valid")

	var previousSettlements map[string]map[string]bee.SentReceived
	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload file to random node
		uIndex := rnd.Intn(c.Size())
		uNode := sortedNodes[uIndex]
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
		if err := ng.NodeClient(uNode).UploadFile(ctx, &file, false); err != nil {
			return fmt.Errorf("node %s: %w", uNode, err)
		}
		fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[uNode].String())

		// validate settlements after uploading a file
		previousSettlements = settlements
		for t := 0; t < 5; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = ng.Balances(ctx)
			if err != nil {
				return err
			}

			settlements, err = ng.Settlements(ctx)
			if err != nil {
				return err
			}
			if err := settlementsHaveHappened(settlements, previousSettlements); err != nil {
				return err
			}

			err = validateSettlements(o.Threshold, overlays, balances, settlements)
			if err != nil {
				fmt.Printf("Invalid settlements after uploading a file: %s\n", err.Error())
				fmt.Println("Retrying ...")
				continue
			}

			fmt.Println("Settlements are valid")
			break
		}

		time.Sleep(time.Duration(o.WaitBeforeDownload) * time.Second)
		// download file from random node
		dIndex := randomIndex(rnd, c.Size(), uIndex)
		dNode := sortedNodes[dIndex]
		size, hash, err := ng.NodeClient(dNode).DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %s: %w", dNode, err)
		}
		if !bytes.Equal(file.Hash(), hash) {
			return fmt.Errorf("File %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.Address().String(), overlays[dNode].String(), file.Size(), size)
		}
		fmt.Printf("File %s downloaded successfully from node %s\n", file.Address().String(), overlays[dNode].String())

		// validate settlements after downloading a file
		previousSettlements = settlements
		for t := 0; t < 5; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = ng.Balances(ctx)
			if err != nil {
				return err
			}

			settlements, err = ng.Settlements(ctx)
			if err != nil {
				return err
			}
			if err := settlementsHaveHappened(settlements, previousSettlements); err != nil {
				return err
			}

			err = validateSettlements(o.Threshold, overlays, balances, settlements)
			if err != nil {
				fmt.Printf("Invalid settlements after downloading a file: %s\n", err.Error())
				fmt.Println("Retrying ...")
				continue
			}

			fmt.Println("Settlements are valid")
			break
		}
	}

	return
}

// DryRunCheck executes settlements validation check without files uploading/downloading
func DryRunCheck(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()

	ng := c.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	balances, err := ng.Balances(ctx)
	if err != nil {
		return err
	}

	settlements, err := ng.Settlements(ctx)
	if err != nil {
		return err
	}

	if err := validateSettlements(o.Threshold, overlays, balances, settlements); err != nil {
		return fmt.Errorf("invalid settlements")
	}
	fmt.Println("Settlements are valid")

	return
}

// validateSettlements checks if settlements are valid
func validateSettlements(threshold int64, overlays bee.NodeGroupOverlays, balances bee.NodeGroupBalances, settlements bee.NodeGroupSettlements) (err error) {
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
		fmt.Println("invalid settlements: no symmetry")
	}

	return
}

// settlementsHaveHappened checks if settlements have happened
func settlementsHaveHappened(current, previous map[string]map[string]bee.SentReceived) error {
	for node, v := range current {
		for peer, settlement := range v {
			if settlement.Received != previous[node][peer].Received || settlement.Sent != previous[node][peer].Sent {
				fmt.Println("Settlements have happened")
				return nil
			}
		}
	}

	return fmt.Errorf("settlements have not happened")
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
