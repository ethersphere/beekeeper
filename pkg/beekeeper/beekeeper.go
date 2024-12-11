package beekeeper

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// Action defines Beekeeper Action's interface. An action that
// needs to expose metrics should implement the metrics.Reporter
// interface.
type Action interface {
	Run(ctx context.Context, cluster orchestration.Cluster, o interface{}) (err error)
}
