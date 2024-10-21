package cashout

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// TODO: remove need for node group, use whole cluster instead

// Options represents settlements check options
type Options struct {
	NodeGroup string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		NodeGroup: "bee",
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	logger logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

type CashoutAction struct {
	node            string
	peer            swarm.Address
	uncashedAmount  *big.Int
	transactionHash string
	oldBalance      *big.Int
}

// Check executes settlements check
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	ng, err := cluster.NodeGroup(o.NodeGroup)
	if err != nil {
		return err
	}
	sortedNodes := ng.NodesSorted()
	var actions []CashoutAction

	for _, node := range sortedNodes {
		client, err := ng.NodeClient(node)
		if err != nil {
			return err
		}
		settlements, err := client.Settlements(ctx)
		if err != nil {
			return err
		}

		for _, peerSettlements := range settlements.Settlements {
			if peerSettlements.Received > 0 {
				peerOverlay, err := swarm.ParseHexAddress(peerSettlements.Peer)
				if err != nil {
					return err
				}
				cashoutStatus, err := client.CashoutStatus(ctx, peerOverlay)
				if err != nil {
					return err
				}

				if cashoutStatus.UncashedAmount.Cmp(big.NewInt(0)) > 0 {
					chequebookBalance, err := client.ChequebookBalance(ctx)
					if err != nil {
						return err
					}

					txHash, err := client.Cashout(ctx, peerOverlay)
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

					c.logger.Infof("cashing out for peer %s on %s in transaction %s", peerOverlay, node, txHash)
				}
			}
		}
	}

LOOP:
	for i := 0; i < 10; i++ {
		time.Sleep(5 * time.Second)

		for _, action := range actions {
			client, err := ng.NodeClient(action.node)
			if err != nil {
				return err
			}
			cashoutStatus, err := client.CashoutStatus(ctx, action.peer)
			if err != nil {
				return err
			}

			if cashoutStatus.Result == nil {
				c.logger.Infof("transaction %s not yet confirmed", action.transactionHash)
				continue LOOP
			}

			if cashoutStatus.Result.Bounced {
				return fmt.Errorf("bouncing cheque on %s from peer %s", action.node, action.peer)
			}

			chequebookBalance, err := client.ChequebookBalance(ctx)
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
