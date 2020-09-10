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

	for i := 0; i < o.UploadNodeCount; i++ {
		// validate balances before uploading a file
		b, err := c.Balances(ctx)
		if err != nil {
			return err
		}
		if err := validateBalances(overlays, b); err != nil {
			return fmt.Errorf("invalid balances before uploading a file")
		}

		// upload file to random node
		uIndex := rnd.Intn(c.Size())
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
		if err := c.Nodes[uIndex].UploadFile(ctx, &file); err != nil {
			return fmt.Errorf("node %d: %w", uIndex, err)
		}
		fmt.Printf("Node %d. File uploaded successfully. Node: %s File: %s\n", uIndex, overlays[uIndex].String(), file.Address().String())

		// validate balances after uploading a file
		b, err = c.Balances(ctx)
		if err != nil {
			return err
		}
		if err := validateBalances(overlays, b); err != nil {
			return fmt.Errorf("invalid balances after uploading a file")
		}

		// download file from random node
		dIndex := randomIndex(rnd, c.Size(), uIndex)
		size, hash, err := c.Nodes[dIndex].DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %d: %w", dIndex, err)
		}
		if !bytes.Equal(file.Hash(), hash) {
			return fmt.Errorf("Node %d. File not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s", dIndex, file.Size(), size, overlays[dIndex].String(), file.Address().String())
		}
		fmt.Printf("Node %d. File downloaded successfully. Node: %s File: %s\n", dIndex, overlays[dIndex].String(), file.Address().String())

		b, err = c.Balances(ctx)
		if err != nil {
			return err
		}
		if err := validateBalances(overlays, b); err != nil {
			return fmt.Errorf("invalid balances after downloading a file")
		}
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
	fmt.Printf("valid balances\n")

	return
}

// validateBalances checks if balances are valid
func validateBalances(overlays []swarm.Address, balances []bee.Balances) (err error) {
	ob := make(map[string]map[string]int)
	for i := 0; i < len(overlays); i++ {
		tmp := make(map[string]int)
		for _, b := range balances[i].Balances {
			tmp[b.Peer] = b.Balance
		}
		ob[overlays[i].String()] = tmp
	}

	// check balance symmetry
	var noSymmetry bool
	for node, v := range ob {
		for peer, balance := range v {
			diff := balance + ob[peer][node]
			if diff != 0 {
				fmt.Printf("Node %s has balance %d with peer %s\n", node, balance, peer)
				fmt.Printf("Peer %s has balance %d with node %s\n", peer, ob[peer][node], node)
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
