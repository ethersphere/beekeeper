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

func RegisterCollectors(p *push.Pusher, c ...prometheus.Collector) {
	for _, cc := range c {
		p.Collector(cc)
	}
}

func PrometheusCollectorsFromFields(i interface{}) (cs []prometheus.Collector) {
	v := reflect.Indirect(reflect.ValueOf(i))
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).CanInterface() {
			continue
		}
		if u, ok := v.Field(i).Interface().(prometheus.Collector); ok {
			cs = append(cs, u)
		}
	}
	return cs
}
