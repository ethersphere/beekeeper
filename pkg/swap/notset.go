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

// sendETH makes ETH deposit
func (n *NotSet) SendETH(ctx context.Context, to string, ammount float64) (tx string, err error) {
	return "", ErrNotSet
}

// sendBZZ makes BZZ token deposit
func (n *NotSet) SendBZZ(ctx context.Context, to string, ammount float64) (tx string, err error) {
	return "", ErrNotSet
}
