package swap

import (
	"context"
)

const (
	BzzDeposit      = 100.0
	BzzGasPrice     = 65536
	BzzTokenAddress = "0x6aab14fe9cccd64a502d23842d916eb5321c26e7"
	EthAccount      = "0x62cab2b3b55f341f10348720ca18063cdb779ad5"
	EthDeposit      = 1.0
	EthGasPrice     = 21000
)

// Client defines Client interface
type Client interface {
	SendETH(ctx context.Context, to string, amount float64) (tx string, err error)
	SendBZZ(ctx context.Context, to string, amount float64) (tx string, err error)
}
