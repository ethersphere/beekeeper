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
		fmt.Println("settlements", s)
		if err := validateSettlements(o.Threshold, overlays, b, s); err != nil {
			return fmt.Errorf("invalid settlements before uploading a file: %s", err.Error())
		}

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
		fmt.Println("settlements", s)
		if err := validateSettlements(o.Threshold, overlays, b, s); err != nil {
			return fmt.Errorf("invalid settlements after uploading a file: %s", err.Error())
		}

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
		fmt.Println("settlements", s)
		if err := validateSettlements(o.Threshold, overlays, b, s); err != nil {
			return fmt.Errorf("invalid settlements after downloading a file: %s", err.Error())
		}
	}

	return
}

// DryRunCheck executes settlements validation check without files uploading/downloading
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

// validateSettlements checks if balances are valid
func validateSettlements(threshold int, overlays []swarm.Address, clusterBalances []bee.Balances, clusterSettlements []bee.Settlements) (err error) {
	// threshold validation
	for i, bs := range clusterBalances {
		for _, b := range bs.Balances {
			if b.Balance > threshold {
				return fmt.Errorf("threshold %d (max %d) exceeded on node %s with peer %s", b.Balance, threshold, overlays[i], b.Peer)

			}
		}
	}

	for i, ss := range clusterSettlements {
		for j, s := range ss.Settlements {
			fmt.Println(i, j, s)
		}
	}

	return
}

// validateBalances checks if balances are valid
func validateBalances(overlays []swarm.Address, clusterBalances []bee.Balances) (err error) {
	balances := make(map[string]map[string]int)
	for i := 0; i < len(overlays); i++ {
		tmp := make(map[string]int)
		for _, b := range clusterBalances[i].Balances {
			tmp[b.Peer] = b.Balance
		}
		balances[overlays[i].String()] = tmp
	}

	// check balance symmetry
	var noSymmetry bool
	for node, v := range balances {
		for peer, balance := range v {
			diff := balance + balances[peer][node]
			fmt.Printf("Node %s has symmetric balance with peer %s\n", node, peer)
			if diff != 0 {
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
