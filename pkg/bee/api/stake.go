package api

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethersphere/beekeeper/pkg/bigint"
)

// StakingService represents Bee's staking service
type StakingService service

type getStakeResponse struct {
	WithdrawableStake *bigint.BigInt `json:"withdrawableStake"`
}
type stakeDepositResponse struct {
	TxHash string `json:"txhash"`
}
type stakeWithdrawResponse struct {
	TxHash string `json:"txhash"`
}

// DepositStake deposits stake
func (s *StakingService) DepositStake(ctx context.Context, amount *big.Int) (txHash string, err error) {
	r := new(stakeDepositResponse)
	err = s.client.requestJSON(ctx, http.MethodPost, fmt.Sprintf("/stake/%d", amount), nil, r)
	if err != nil {
		return "", err
	}
	return r.TxHash, nil
}

// GetWithdrawableStake gets stake
func (s *StakingService) GetWithdrawableStake(ctx context.Context) (withdrawableStake *big.Int, err error) {
	r := new(getStakeResponse)
	err = s.client.requestJSON(ctx, http.MethodGet, "/stake", nil, r)
	if err != nil {
		return nil, err
	}
	return r.WithdrawableStake.Int, nil
}

// MigrateStake withdraws stake
func (s *StakingService) MigrateStake(ctx context.Context) (txHash string, err error) {
	r := new(stakeWithdrawResponse)
	err = s.client.requestJSON(ctx, http.MethodDelete, "/stake", nil, r)
	if err != nil {
		return "", err
	}
	return r.TxHash, nil
}
