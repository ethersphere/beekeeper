package redistribution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Options represents stake options
type Options struct {
	ContractAddr     string
	CallerPrivateKey string
	GethURL          string
	GethChainID      *big.Int
	FromBlock        uint64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		GethChainID: big.NewInt(12345),
	}
}

// compile check whether EventsCheck implements interface
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

// Run executes ping check
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) (err error) {
	opts, ok := o.(Options)
	if !ok {
		return err
	}

	contract, err := NewContract(opts)
	if err != nil {
		return fmt.Errorf("new contract: %w", err)
	}

	fromBlock := opts.FromBlock

	var i int
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			c.logger.Infof("starting iteration: #%d (from block: %d)", i, fromBlock)
		}

	FilterCountCommits:
		it, err := contract.FilterCountCommits(&bind.FilterOpts{
			Context: ctx,
			Start:   fromBlock,
		})

		if err != nil {
			return fmt.Errorf("filter count commits: %w", err)
		}

		for it.Next() {
			c.logger.Info("count commits: fast forward...")
		}

		if it.Event == nil {
			c.logger.Info("count commits: no events found, retrying...")
			time.Sleep(time.Second)
			goto FilterCountCommits
		}

		fromBlock = it.Event.Raw.BlockNumber
		commitsCount := it.Event.Count

	FilterCountReveals:
		crit, err := contract.FilterCountReveals(&bind.FilterOpts{
			Context: ctx,
			Start:   fromBlock - 1,
		})

		if err != nil {
			return fmt.Errorf("filter count commits:%w", err)
		}

		for crit.Next() {
			c.logger.Info("count reveals: fast forward...")
		}

		if crit.Event == nil {
			c.logger.Info("count reveals: no event found, retrying...")
			time.Sleep(5 * time.Second)
			goto FilterCountReveals
		}

		revealsCount := crit.Event.Count
		if commitsCount.Cmp(revealsCount) != 0 {
			c.logger.Errorf("want reveals: %d, got %d", commitsCount.Int64(), revealsCount.Int64())
		} else {
			c.logger.Infof("finished iteration %d, got equal commit/reveals: %d", i, commitsCount.Int64())
		}

		i++
	}
}

func NewContract(opts Options) (*Redistribution, error) {
	if opts.GethURL == "" {
		panic(errors.New("geth URL not provided"))
	}

	geth, err := ethclient.Dial(opts.GethURL)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	addr := common.HexToAddress(opts.ContractAddr)
	contract, err := NewRedistribution(addr, geth)
	if err != nil {
		return nil, fmt.Errorf("new contract instance: %w", err)
	}

	return contract, nil
}
