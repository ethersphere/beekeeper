package balances

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	test "github.com/ethersphere/beekeeper/pkg/test"
)

// Options represents check options
type Options struct {
	DryRun             bool
	FileName           string
	FileSize           int64
	GasPrice           string
	PostageAmount      int64
	PostageDepth       uint64
	PostageLabel       string
	Seed               int64
	UploadNodeCount    int
	WaitBeforeDownload time.Duration
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		DryRun:             false,
		FileName:           "balances",
		FileSize:           1 * 1024 * 1024, // 1mb,
		GasPrice:           "",
		PostageAmount:      1,
		PostageDepth:       16,
		PostageLabel:       "test-label",
		Seed:               0,
		UploadNodeCount:    1,
		WaitBeforeDownload: 5 * time.Second,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		fmt.Println("running balances (dry mode)")
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running balances")

	var checkCase *test.CheckCase

	caseOpts := test.CaseOptions{
		FileName:      o.FileName,
		FileSize:      o.FileSize,
		GasPrice:      o.GasPrice,
		PostageAmount: o.PostageAmount,
		PostageDepth:  o.PostageDepth,
		PostageLabel:  o.PostageLabel,
		Seed:          o.Seed,
	}

	if checkCase, err = test.NewCheckCase(ctx, cluster, caseOpts); err != nil {
		return err
	}

	balances, err := checkCase.Balances(ctx)
	if err != nil {
		return err
	}

	// initial validation
	if err := validateBalances(balances); err != nil {
		return fmt.Errorf("invalid initial balances: %v", err)
	}

	fmt.Println("Balances are valid")

	// repeats
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload/check
		bee := checkCase.RandomBee()

		file, err := bee.UploadRandomFile(ctx)
		if err != nil {
			return err
		}

		newBalances, err := checkCase.Balances(ctx)
		if err != nil {
			return err
		}

		if err := expectBalancesHaveChanged(ctx, balances, newBalances); err != nil {
			return err
		}

		balances = newBalances

		// download/check
		if err := checkCase.RandomBee().ExpectToHaveFile(ctx, file); err != nil {
			return err
		}

		newBalances, err = checkCase.Balances(ctx)
		if err != nil {
			return err
		}

		if err := expectBalancesHaveChanged(ctx, balances, newBalances); err != nil {
			return err
		}
	}

	return nil
}

// dryRun executes balances validation check without files uploading/downloading
func dryRun(ctx context.Context, cluster orchestration.Cluster, o Options) (err error) {
	balances, err := cluster.Balances(ctx)
	if err != nil {
		return err
	}

	flatBalances := flattenBalances(balances)
	if err := validateBalances(flatBalances); err != nil {
		return fmt.Errorf("invalid balances")
	}

	fmt.Println("Balances are valid")

	return
}

func expectBalancesHaveChanged(ctx context.Context, balances, newBalances orchestration.NodeGroupBalances) error {
	for t := 0; t < 5; t++ {
		time.Sleep(2 * time.Duration(t) * time.Second)

		balancesHaveChanged(newBalances, balances)

		if err := validateBalances(newBalances); err != nil {
			fmt.Println("Invalid balances after downloading a file:", err)
			fmt.Println("Retrying ...", t)
			continue
		}

		fmt.Println("Balances are valid")

		break
	}

	return nil
}

// validateBalances checks balances symmetry
func validateBalances(balances map[string]map[string]int64) (err error) {
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
func balancesHaveChanged(current, previous orchestration.NodeGroupBalances) {
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

// flattenBalances convenience function
func flattenBalances(b orchestration.ClusterBalances) map[string]map[string]int64 {
	res := make(map[string]map[string]int64)
	for _, ngb := range b {
		for n, balances := range ngb {
			res[n] = balances
		}
	}
	return res
}
