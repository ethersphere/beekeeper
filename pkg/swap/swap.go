package swap

import (
	"context"
)

const (
	BzzTokenAddress       = "0x6aab14fe9cccd64a502d23842d916eb5321c26e7"
	EthAccount            = "0x62cab2b3b55f341f10348720ca18063cdb779ad5"
	GasPrice        int64 = 10000000000
	BzzGasLimit           = 100000
	EthGasLimit           = 21000
	mintBzz               = "0x40c10f19"
	transferBzz           = "0xa9059cbb"
)

// Client defines Client interface
type Client interface {
	SendETH(ctx context.Context, to string, amount float64) (tx string, err error)
	SendBZZ(ctx context.Context, to string, amount float64) (tx string, err error)
	SendGBZZ(ctx context.Context, to string, amount float64) (tx string, err error)
	AttestOverlayEthAddress(ctx context.Context, ethAddr string) (tx string, err error)
}
