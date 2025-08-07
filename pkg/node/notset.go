package node

import (
	"context"
	"errors"
)

// ErrNotSet represents error when NodeClient is not set
var ErrNotSet = errors.New("node client not initialized")

// compile check whether NotSet implements NodeProvider interface
var _ NodeProvider = (*NotSet)(nil)

type NotSet struct{}

// GetNodes implements NodeProvider.
func (n *NotSet) GetNodes(ctx context.Context) (NodeList, error) {
	return nil, ErrNotSet
}
