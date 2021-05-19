package swap

import (
	"context"
)

// Swap defines Swap interface
type Client interface {
	SendETH(ctx context.Context, from, to string, ammount int64) (err error)
	SendBZZ(ctx context.Context, from, to string, ammount int64) (err error)
}
