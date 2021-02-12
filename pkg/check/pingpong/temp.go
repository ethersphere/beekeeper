package pingpong

import (
	"context"
	"fmt"

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

	opts := Options{
		MetricsEnabled: o.MetricsEnabled,
		MetricsPusher:  o.MetricsPusher,
	}
	if err := CheckD(ctx, cluster, opts); err != nil {
		return err
	}

	fmt.Println("pingpong check completed successfully")
	return
}
