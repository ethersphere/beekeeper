package balances

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	DryRun             bool
	FileName           string
	FileSize           int64
	NodeGroup          string // TODO: support multi node group cluster
	Seed               int64
	UploadNodeCount    int
	WaitBeforeDownload int // seconds
}

var DefaultOptions = Options{
	DryRun:             false,
	FileName:           "balances",
	FileSize:           1 * 1024 * 1024, // 1mb,
	NodeGroup:          "bee",
	Seed:               random.Int64(),
	UploadNodeCount:    1,
	WaitBeforeDownload: 5,
}

// compile check whether Check implements interface
var _ check.Check = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() check.Check {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	fmt.Println("DefaultOptions", DefaultOptions)
	fmt.Println("AppliedOptions", o)
	return
	if o.DryRun {
		fmt.Println("running balances (dry mode)")
		return dryRun(ctx, cluster, o)
	}
	fmt.Println("running balances")

	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	ng := cluster.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	// Initial balances validation
	balances, err := ng.Balances(ctx)
	if err != nil {
		return err
	}
	if err := validateBalances(overlays, balances); err != nil {
		return fmt.Errorf("invalid initial balances: %s", err.Error())
	}
	fmt.Println("Balances are valid")

	var previousBalances bee.NodeGroupBalances
	sortedNodes := ng.NodesSorted()
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload file to random node
		uIndex := rnd.Intn(cluster.Size())
		uNode := sortedNodes[uIndex]
		file := bee.NewRandomFile(rnd, fmt.Sprintf("%s-%d", o.FileName, uIndex), o.FileSize)
		if err := ng.NodeClient(uNode).UploadFile(ctx, &file, false); err != nil {
			return fmt.Errorf("node %s: %w", uNode, err)
		}
		fmt.Printf("File %s uploaded successfully to node %s\n", file.Address().String(), overlays[uNode].String())

		// Validate balances after uploading a file
		previousBalances = balances
		for t := 0; t < 5; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = ng.Balances(ctx)
			if err != nil {
				return err
			}
			balancesHaveChanged(balances, previousBalances)

			err = validateBalances(overlays, balances)
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
		dIndex := randomIndex(rnd, cluster.Size(), uIndex)
		dNode := sortedNodes[dIndex]
		size, hash, err := ng.NodeClient(dNode).DownloadFile(ctx, file.Address())
		if err != nil {
			return fmt.Errorf("node %s: %w", dNode, err)
		}
		if !bytes.Equal(file.Hash(), hash) {
			return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.Address().String(), overlays[dNode].String(), file.Size(), size)
		}
		fmt.Printf("File %s downloaded successfully from node %s\n", file.Address().String(), overlays[dNode].String())

		// Validate balances after downloading a file
		previousBalances = balances
		for t := 0; t < 5; t++ {
			time.Sleep(2 * time.Duration(t) * time.Second)

			balances, err = ng.Balances(ctx)
			if err != nil {
				return err
			}
			balancesHaveChanged(balances, previousBalances)

			err := validateBalances(overlays, balances)
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

// dryRun executes balances validation check without files uploading/downloading
func dryRun(ctx context.Context, cluster *bee.Cluster, o Options) (err error) {
	ng := cluster.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	balances, err := ng.Balances(ctx)
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
func validateBalances(overlays map[string]swarm.Address, balances bee.NodeGroupBalances) (err error) {
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
