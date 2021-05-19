package swap

import (
	"context"
)

const (
	BzzTokenAddress = "0x6aab14fe9cccd64a502d23842d916eb5321c26e7"
	EthAccount      = "0x62cab2b3b55f341f10348720ca18063cdb779ad5"
)

// Swap defines Swap interface
type Client interface {
	SendETH(ctx context.Context, to string, ammount int64) (err error)
	SendBZZ(ctx context.Context, ammount int64) (err error)
}
