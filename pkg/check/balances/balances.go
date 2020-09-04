package balances

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Check executes balances check
func Check(cluster bee.Cluster, pusher *push.Pusher, pushMetrics bool) (err error) {
	fmt.Println("balances")

	return
}
