package pingpong

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
)

// compile check whether Ping implements interface
var _ check.Check = (*Ping)(nil)

// Ping ...
type Ping struct{}

// NewPing ...
func NewPing() *Ping {
	return &Ping{}
}

// Run ...
func (p *Ping) Run(ctx context.Context, cluster *bee.Cluster, o check.Options) (err error) {
	fmt.Println("checking pingpong")

	nodeGroups := cluster.NodeGroups()
	for _, ng := range nodeGroups {
		nodesClients, err := ng.NodesClients(ctx)
		if err != nil {
			return fmt.Errorf("get nodes clients: %w", err)
		}

		for n := range nodeStream(ctx, nodesClients) {
			for t := 0; t < 5; t++ {
				time.Sleep(2 * time.Duration(t) * time.Second)

				if n.Error != nil {
					if t == 4 {
						return fmt.Errorf("node %s: %w", n.Name, n.Error)
					}
					fmt.Printf("node %s: %v\n", n.Name, n.Error)
					continue
				}
				fmt.Printf("Node %s: %s Peer: %s RTT: %s\n", n.Name, n.Address, n.PeerAddress, n.RTT)
				break
			}
		}
	}

	fmt.Println("pingpong check completed successfully")
	return
}
