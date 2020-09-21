package balances

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

// Options represents balances check options
type Options struct {
	UploadNodeCount int
	FileName        string
	FileSize        int64
	Seed            int64
}

// Check executes balances check
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	// Initial balances validation
	balances, err := c.Balances(ctx)
	if err != nil {
		return err
	}
	if err := validateBalances(overlays, balances); err != nil {
		return fmt.Errorf("invalid initial balances: %s", err.Error())
	}
	fmt.Println("Balances are valid")

	var previousBalances map[string]map[string]int
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload file to random node
		uIndex := rnd.Intn(c.Size())
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
		if err := c.Nodes[uIndex].UploadFile(ctx, &file, false); err != nil {
			return fmt.Errorf("node %d: %w", uIndex, err)
		}
		fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[uIndex].String())

		// Validate balances after uploading a file
		previousBalances = balances
		balances, err = c.Balances(ctx)
		if err != nil {
			return err
		}
		balancesHaveChanged(balances, previousBalances)

		if err := validateBalances(overlays, balances); err != nil {
			return fmt.Errorf("invalid balances after uploading a file")
		}
		fmt.Println("Balances are valid")

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

		// Validate balances after downloading a file
		previousBalances = balances
		balances, err = c.Balances(ctx)
		if err != nil {
			return err
		}
		balancesHaveChanged(balances, previousBalances)

		if err := validateBalances(overlays, balances); err != nil {
			return fmt.Errorf("invalid balances after downloading a file")
		}
		fmt.Println("Balances are valid")
	}

	return
}

// DryRunCheck executes balances validation check without files uploading/downloading
func DryRunCheck(c bee.Cluster) (err error) {
	ctx := context.Background()

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	balances, err := c.Balances(ctx)
	if err != nil {
		return err
	}

	if err := validateBalances(overlays, balances); err != nil {
		return fmt.Errorf("invalid balances")
	}
	fmt.Println("Balances are valid")

	return
}

// validateBalances checks balances symmetry
func validateBalances(overlays []swarm.Address, balances map[string]map[string]int) (err error) {
	var noSymmetry bool
	for node, v := range balances {
		for peer, balance := range v {
			diff := balance + balances[peer][node]
			if diff != 0 {
				fmt.Printf("Node %s has asymmetric balance with peer %s\n", node, peer)
				fmt.Printf("Node %s has balance %d with peer %s\n", node, balance, peer)
				fmt.Printf("Peer %s has balance %d with node %s\n", peer, balances[peer][node], node)
				fmt.Printf("Difference: %d\n", diff)
				noSymmetry = true
			}
		}
	}
	if noSymmetry {
		return fmt.Errorf("invalid balances: no symmetry")
	}

	return
}

// balancesHaveChanged checks if balances have changed
func balancesHaveChanged(current, previous map[string]map[string]int) {
	for node, v := range current {
		for peer, balance := range v {
			if balance != previous[node][peer] {
				fmt.Println("Balances have changed")
				return
			}
		}
	}
	fmt.Println("Balances have not changed")
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
