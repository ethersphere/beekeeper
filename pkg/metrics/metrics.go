package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Namespace is prefixed before every metric. If it is changed, it must be done
// before any metrics collector is registered.
var Namespace = "beekeeper"

type Reporter interface {
	Report() []prometheus.Collector
}

func RegisterCollectors(p *push.Pusher, c ...prometheus.Collector) {
	for _, cc := range c {
		p.Collector(cc)
	}
}
