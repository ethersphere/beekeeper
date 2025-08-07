package nuker

import (
	"context"

	v1 "k8s.io/api/apps/v1"
)

var _ NeighborhoodArgProvider = (*NeighborhoodArgProviderNotSet)(nil)

type NeighborhoodArgProviderNotSet struct{}

// GetArgs implements NeighborhoodArgProvider.
func (n *NeighborhoodArgProviderNotSet) GetArgs(ctx context.Context, ss *v1.StatefulSet, restartArgs []string) ([]string, error) {
	return restartArgs, nil
}
