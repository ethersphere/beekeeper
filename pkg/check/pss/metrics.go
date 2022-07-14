package pss

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	SendAndReceiveGauge *prometheus.GaugeVec
}

func newMetrics() metrics {
	subsystem := "check_pss"
	return metrics{
		SendAndReceiveGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "pss_total_duration",
				Help:      "total duration between sending a message and receiving it on the other end.",
			},
			[]string{"nodeA", "nodeB"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
