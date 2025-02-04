package swap

import (
	"context"
	"errors"
)

// ErrNotSet represents error when Swap client is not set
var ErrNotSet = errors.New("swap client not set")

// compile check whether NotSet implements Swap interface
var _ Client = (*NotSet)(nil)

type NotSet struct{}

// SendETH makes ETH deposit
func (n *NotSet) SendETH(ctx context.Context, to string, amount float64) (tx string, err error) {
	return "", ErrNotSet
}

// SendBZZ makes BZZ token deposit
func (n *NotSet) SendBZZ(ctx context.Context, to string, amount float64) (tx string, err error) {
	return "", ErrNotSet
}

// SendGBZZ makes gBZZ token deposit
func (n *NotSet) SendGBZZ(ctx context.Context, to string, amount float64) (tx string, err error) {
	return "", ErrNotSet
}

func (n *NotSet) AttestOverlayEthAddress(ctx context.Context, ethAddr string) (tx string, err error) {
	return "", ErrNotSet
}

// FetchBlockTime(ctx context.Context) (blockTime int64, err error)
func (n *NotSet) FetchBlockTime(ctx context.Context, opts ...Option) (blockTime int64, err error) {
	return 0, ErrNotSet
}
