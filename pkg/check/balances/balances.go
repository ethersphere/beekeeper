package balances

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents balances check options
type Options struct {
	NodeGroup          string
	UploadNodeCount    int
	FileName           string
	FileSize           int64
	Seed               int64
	WaitBeforeDownload int
	PostageAmount      int64
	PostageWait        time.Duration
}

// Check executes balances check
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}

	flatOverlays := flattenOverlays(overlays)

	// Initial balances validation
	balances, err := c.Balances(ctx)
	if err != nil {
		return err
	}

	flatBalances := flattenBalances(balances)
	if err := validateBalances(flatOverlays, flatBalances); err != nil {
		return fmt.Errorf("invalid initial balances: %s", err.Error())
	}
	fmt.Println("Balances are valid")

	var previousBalances bee.NodeGroupBalances
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload file to random node

		ng, nodeName, overlay := overlays.Random(rnd)

		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%s", o.FileName, nodeName), o.FileSize)
		client := c.NodeGroups()[ng].NodeClient(nodeName)

		// add some buffer to ensure depth is enough
		depth := 2 + bee.EstimatePostageBatchDepth(file.Size())
		batchID, err := client.CreatePostageBatch(ctx, o.PostageAmount, depth, "test-label")
		if err != nil {
			return fmt.Errorf("node %s: created batched id %w", nodeName, err)
		}

		fmt.Printf("node %s: created batched id %s\n", nodeName, batchID)

		time.Sleep(o.PostageWait)

		if err := client.UploadFile(ctx, &file, api.UploadOptions{BatchID: batchID}); err != nil {
			return fmt.Errorf("node %s: %w", nodeName, err)
		}
		fmt.Printf("File %s uploaded successfully to node \"%s\" (%s)\n", file.Address().String(), nodeName, overlay.String())

		// Validate balances after uploading a file
		previousBalances = flatBalances
		for t := 0; t < 5; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = c.Balances(ctx)
			if err != nil {
				return err
			}
			flatBalances = flattenBalances(balances)
			balancesHaveChanged(flatBalances, previousBalances)

			err = validateBalances(flatOverlays, flatBalances)
			if err != nil {
				fmt.Printf("Invalid balances after uploading a file: %s\n", err.Error())
				fmt.Println("Retrying ...")
				continue
			}

			fmt.Println("Balances are valid")
			break
		}

		time.Sleep(time.Duration(o.WaitBeforeDownload) * time.Second)
		// download file from random node
		ng, nodeName, overlay = overlays.Random(rnd)
		size, hash, err := c.NodeGroups()[ng].NodeClient(nodeName).DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %s: %w", nodeName, err)
		}
		if !bytes.Equal(file.Hash(), hash) {
			return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.Address().String(), overlay.String(), file.Size(), size)
		}
		fmt.Printf("File %s downloaded successfully from node \"%s\"(%s)\n", file.Address().String(), nodeName, overlay.String())

		// Validate balances after downloading a file
		previousBalances = flatBalances
		for t := 0; t < 5; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = c.Balances(ctx)
			if err != nil {
				return err
			}
			flatBalances = flattenBalances(balances)
			balancesHaveChanged(flatBalances, previousBalances)

			err := validateBalances(flatOverlays, flatBalances)
			if err != nil {
				fmt.Printf("Invalid balances after downloading a file: %s\n", err.Error())
				fmt.Println("Retrying ...")
				continue
			}

			fmt.Println("Balances are valid")
			break
		}
	}

	return
}

// DryRunCheck executes balances validation check without files uploading/downloading
func DryRunCheck(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()

	overlays, err := c.Overlays(ctx)
	if err != nil {
		return err
	}
	flatOverlays := flattenOverlays(overlays)

	balances, err := c.Balances(ctx)
	if err != nil {
		return err
	}
	flatBalances := flattenBalances(balances)

	if err := validateBalances(flatOverlays, flatBalances); err != nil {
		return fmt.Errorf("invalid balances")
	}
	fmt.Println("Balances are valid")

	return
}

// validateBalances checks balances symmetry
func validateBalances(overlays map[string]swarm.Address, balances map[string]map[string]int64) (err error) {
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
func balancesHaveChanged(current, previous bee.NodeGroupBalances) {
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

func flattenOverlays(o bee.ClusterOverlays) map[string]swarm.Address {
	res := make(map[string]swarm.Address)
	for _, ngo := range o {
		for n, over := range ngo {
			res[n] = over
		}
	}
	return res
}

func flattenBalances(b bee.ClusterBalances) map[string]map[string]int64 {
	res := make(map[string]map[string]int64)
	for _, ngb := range b {
		for n, balances := range ngb {
			res[n] = balances
		}
	}
	return res
}
