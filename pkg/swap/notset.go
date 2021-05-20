package swap

import (
	"context"
	"errors"
	"math/big"
)

// ErrNotSet represents error when Swap client is not set
var ErrNotSet = errors.New("swap client not set")

type NotSet struct{}

// sendETH makes ETH deposit
func (n *NotSet) SendETH(ctx context.Context, to string, ammount *big.Int) (tx string, err error) {
	return "", ErrNotSet
}

// sendBZZ makes BZZ token deposit
func (n *NotSet) SendBZZ(ctx context.Context, to string, ammount *big.Int) (tx string, err error) {
	return "", ErrNotSet
}
