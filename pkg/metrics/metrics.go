package metrics

import (
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Namespace is prefixed before every metric. If it is changed, it must be done
// before any metrics collector is registered.
var Namespace = "beekeeper"

type Reporter interface {
	Report() []prometheus.Collector
}

func PrometheusCollectorsFromFields(i any) (cs []prometheus.Collector) {
	v := reflect.Indirect(reflect.ValueOf(i))
	for _, field := range v.Fields() {
		if !field.CanInterface() {
			continue
		}
		if u, ok := field.Interface().(prometheus.Collector); ok {
			cs = append(cs, u)
		}
	}
	return cs
}

func RegisterCollectors(p *push.Pusher, c ...prometheus.Collector) {
	for _, cc := range c {
		p.Collector(cc)
	}
}
