package balances

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
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
	PostageLabel       string
	PostageWait        time.Duration
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
		PostageLabel:       "test-label",
		PostageWait:        5 * time.Second,
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
		PostageLabel:  o.PostageLabel,
		PostageWait:   o.PostageWait,
		Seed:          o.Seed,
	}

	if checkCase, err = test.NewCheckCase(ctx, cluster, caseOpts); err != nil {
		return err
	}

	if err := checkCase.SaveBalances(); err != nil {
		return err
	}

	if err := checkCase.ExpectValidInitialBalances(ctx); err != nil {
		return err
	}

	// repeats
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload/check
		node := checkCase.RandomNode()

		file, err := node.UploadRandomFile(ctx)

		if err != nil {
			return err
		}
		if err := checkCase.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}

		// download/check
		if err := checkCase.RandomNode().ExpectToHaveFile(ctx, file); err != nil {
			return err
		}
		if err := checkCase.SaveBalances(); err != nil {
			return err
		}
		if err := checkCase.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}
	}

	return nil
}

// dryRun executes balances validation check without files uploading/downloading
func dryRun(ctx context.Context, cluster orchestration.Cluster, o Options) (err error) {
	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return err
	}
	flatOverlays := flattenOverlays(overlays)

	balances, err := cluster.Balances(ctx)
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

func flattenOverlays(o orchestration.ClusterOverlays) map[string]swarm.Address {
	res := make(map[string]swarm.Address)
	for _, ngo := range o {
		for n, over := range ngo {
			res[n] = over
		}
	}
	return res
}

func flattenBalances(b orchestration.ClusterBalances) map[string]map[string]int64 {
	res := make(map[string]map[string]int64)
	for _, ngb := range b {
		for n, balances := range ngb {
			res[n] = balances
		}
	}
	return res
}
