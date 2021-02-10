package pingpong

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
)

// compile check whether client implements interface
var _ check.Check = (*Ping)(nil)

// Ping ...
type Ping struct{}

// NewPing ...
func NewPing() *Ping {
	return &Ping{}
}

// Run ...
func (p *Ping) Run(ctx context.Context, cluster *bee.Cluster, o check.Options) (err error) {
	fmt.Println("ping pong\nping pong\nping pong")
	return
}
