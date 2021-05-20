package swap

import (
	"context"
	"math/big"
)

const (
	BzzDeposit      = "1000000000000000000"
	BzzTokenAddress = "0x6aab14fe9cccd64a502d23842d916eb5321c26e7"
	EthAccount      = "0x62cab2b3b55f341f10348720ca18063cdb779ad5"
	EthDepost       = "1000000000000000000"
)

// Swap defines Swap interface
type Client interface {
	SendETH(ctx context.Context, to string, ammount *big.Int) (tx string, err error)
	SendBZZ(ctx context.Context, to string, ammount *big.Int) (tx string, err error)
}
