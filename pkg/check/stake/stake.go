package stake

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethersphere/beekeeper/pkg/bee/api"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents stake options
type Options struct {
	Amount             *big.Int
	InsufficientAmount *big.Int
	ContractAddr       string
	CallerPrivateKey   string
	GethURL            string
	GethChainID        *big.Int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Amount:             big.NewInt(100000000000000000),
		InsufficientAmount: big.NewInt(102400),
		GethChainID:        big.NewInt(12345),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

var zero = big.NewInt(0)

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return err
	}

	s, geth, err := newStake(o)
	if err != nil {
		return fmt.Errorf("new stakeing: %w", err)
	}

	stake, err := newSession(s, geth, o)
	if err != nil {
		return fmt.Errorf("new staking contract session: %w", err)
	}

	if paused, err := stake.Paused(); err != nil {
		return fmt.Errorf("check if contract is paused: %w", err)
	} else if paused {
		c.logger.Info("contract is paused, skipping")
		return nil
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.FullNodeNames()
	node := sortedNodes[1]
	c.logger.Infof("checking stake for node %s", node)
	client := clients[node]

	if err := expectStakeAmountIs(ctx, client, zero); err != nil {
		return err
	}

	// depositing insufficient amount should fail
	_, err = client.DepositStake(ctx, o.InsufficientAmount)

	if !api.IsHTTPStatusErrorCode(err, 400) {
		return fmt.Errorf("deposit insufficient stake amount: expected code %v, got %v", 400, err)
	}

	if err := expectStakeAmountIs(ctx, client, zero); err != nil {
		return err
	}

	// depositing sufficient amount should succeed
	_, err = client.DepositStake(ctx, o.Amount)
	if err != nil {
		return fmt.Errorf("initial stake deposit: %w", err)
	}

	if err := expectStakeAmountIs(ctx, client, o.Amount); err != nil {
		return err
	}

	// should allow increasing the stake amount
	stakedAmount := new(big.Int).Add(o.Amount, big.NewInt(1))

	_, err = client.DepositStake(ctx, big.NewInt(1))
	if err != nil {
		return fmt.Errorf("increase stake amount: %w", err)
	}

	if err := expectStakeAmountIs(ctx, client, stakedAmount); err != nil {
		return err
	}

	// should not allow withdrawing from a running contract
	_, err = client.MigrateStake(ctx)
	if err == nil {
		return errors.New("withdraw from running contract should fail")
	}

	if err := expectStakeAmountIs(ctx, client, stakedAmount); err != nil {
		return err
	}

	tx, err := stake.Pause()
	if err != nil {
		return fmt.Errorf("pause contract: %w", err)
	}

	_, err = bind.WaitMined(ctx, geth, tx)
	if err != nil {
		return fmt.Errorf("watch tx: %w", err)
	}

	defer func() {
		if _, err := stake.UnPause(); err != nil {
			c.logger.Errorf("unpause contract: %v", err)
		}
	}()

	// successful withdraw should set the staked amount to 0
	_, err = client.MigrateStake(ctx)
	if err != nil {
		return fmt.Errorf("withdraw from paused contract: %w", err)
	}

	if err := expectStakeAmountIs(ctx, client, zero); err != nil {
		return err
	}

	return nil
}

func expectStakeAmountIs(ctx context.Context, client *bee.Client, expected *big.Int) error {
	current, err := client.GetStake(ctx)
	if err != nil {
		return fmt.Errorf("get stake amount: %w", err)
	}

	if current.Cmp(expected) != 0 {
		return fmt.Errorf("expected stake amount to be %d, got: %d", expected, current)
	}

	withdrawable, err := client.GetWithdrawableStake(ctx)
	if err != nil {
		return fmt.Errorf("get stake amount: %w", err)
	}

	if withdrawable.Cmp(expected) != 0 {
		return fmt.Errorf("expected stake amount to be %d, got: %d", expected, withdrawable)
	}

	return nil
}
