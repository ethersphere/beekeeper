package bee

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

func (c *CheckCase) ExpectValidInitialBalances(ctx context.Context) error {
	flatBalances := c.prevBalances()
	flatOverlays := flattenOverlays(c.overlays)

	if err := expectCreditsEqualDebits(flatOverlays, flatBalances); err != nil {
		return fmt.Errorf("invalid initial balances: %s", err.Error())
	}

	fmt.Println("Balances are valid")

	return nil
}

func (c *CheckCase) ExpectBalancesHaveChanged(ctx context.Context) error {
	for t := 0; t < 5; t++ {
		time.Sleep(2 * time.Duration(t) * time.Second)

		balances, err := c.cluster.Balances(c.ctx)

		if err != nil {
			return err
		}

		flatBalances := flattenBalances(balances)
		balancesHaveChanged(flatBalances, c.prevBalances())

		flatOverlays := flattenOverlays(c.overlays)

		if err := expectCreditsEqualDebits(flatOverlays, flatBalances); err != nil {
			fmt.Println("Invalid balances after downloading a file:", err)
			fmt.Println("Retrying ...", t)
			continue
		}

		fmt.Println("Balances are valid")

		break
	}

	return nil
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

func expectCreditsEqualDebits(overlays map[string]swarm.Address, balances map[string]map[string]int64) (err error) {
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
