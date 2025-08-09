package nuker

import (
	"context"
)

var _ NeighborhoodArgProvider = (*NeighborhoodArgProviderNotSet)(nil)

type NeighborhoodArgProviderNotSet struct{}

// GetArgs implements NeighborhoodArgProvider.
func (n *NeighborhoodArgProviderNotSet) GetArgs(ctx context.Context, nodeName string, restartArgs []string) ([]string, error) {
	return restartArgs, nil
}
