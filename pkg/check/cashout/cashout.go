package cashout

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

// Options represents settlements check options
type Options struct {
	NodeGroup string
}

type CashoutAction struct {
	node            string
	peer            swarm.Address
	uncashedAmount  *big.Int
	transactionHash string
	oldBalance      *big.Int
}

// Check executes settlements check
func Check(c *bee.Cluster, o Options) (err error) {
	ctx := context.Background()

	ng := c.NodeGroup(o.NodeGroup)

	sortedNodes := ng.NodesSorted()
	var actions []CashoutAction

	for _, node := range sortedNodes {
		settlements, err := ng.NodeClient(node).Settlements(ctx)
		if err != nil {
			return err
		}

		for _, peerSettlements := range settlements.Settlements {
			if peerSettlements.Received > 0 {
				peerOverlay, err := swarm.ParseHexAddress(peerSettlements.Peer)
				if err != nil {
					return err
				}
				cashoutStatus, err := ng.NodeClient(node).CashoutStatus(ctx, peerOverlay)
				if err != nil {
					return err
				}

				if cashoutStatus.UncashedAmount.Cmp(big.NewInt(0)) > 0 {
					chequebookBalance, err := ng.NodeClient(node).ChequebookBalance(ctx)
					if err != nil {
						return err
					}

					txHash, err := ng.NodeClient(node).Cashout(ctx, peerOverlay)
					if err != nil {
						return err
					}

					actions = append(actions, CashoutAction{
						node:            node,
						peer:            peerOverlay,
						uncashedAmount:  cashoutStatus.UncashedAmount,
						transactionHash: txHash,
						oldBalance:      chequebookBalance.TotalBalance,
					})

					fmt.Printf("cashing out for peer %s on %s in transaction %s\n", peerOverlay, node, txHash)
				}
			}
		}
	}

LOOP:
	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Second)

		for _, action := range actions {
			cashoutStatus, err := ng.NodeClient(action.node).CashoutStatus(ctx, action.peer)
			if err != nil {
				return err
			}

			if cashoutStatus.Result == nil {
				fmt.Printf("transaction %s not yet confirmed\n", action.transactionHash)
				continue LOOP
			}

			if cashoutStatus.Result.Bounced {
				return fmt.Errorf("bouncing cheque on %s from peer %s", action.node, action.peer)
			}

			chequebookBalance, err := ng.NodeClient(action.node).ChequebookBalance(ctx)
			if err != nil {
				return err
			}

			if action.oldBalance.Cmp(chequebookBalance.TotalBalance) == 0 {
				return fmt.Errorf("chequebook balance not changed after cashout. was %d, now is %d", action.oldBalance, chequebookBalance.TotalBalance)
			}
		}

		return nil
	}

	return errors.New("not all cashouts confirmed")
}
