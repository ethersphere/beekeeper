package debugapi

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
)

// StakingService represents Bee's staking service
type StakingService service

type getStakeResponse struct {
	StakedAmount *big.Int `json:"stakedAmount"`
}
type stakeDepositResponse struct {
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

// GetStakedAmount gets stake
func (s *StakingService) GetStakedAmount(ctx context.Context) (stakedAmount *big.Int, err error) {
	r := new(getStakeResponse)
	err = s.client.requestJSON(ctx, http.MethodGet, "/stake", nil, r)
	if err != nil {
		return nil, err
	}
	return r.StakedAmount, nil
}
