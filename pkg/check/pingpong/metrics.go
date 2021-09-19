package pingpong

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	RttGauge     *prometheus.GaugeVec
	RttHistogram prometheus.Histogram
}

func newMetrics() metrics {
	subsystem := "check_pingpong"
	return metrics{
		RttGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "rtt_duration_seconds",
				Help:      "Ping round-trip time duration Gauge",
			},
			[]string{"node", "peer"},
		),
		RttHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "rtt_seconds",
				Help:      "Ping round-trip time duration Histogram",
				Buckets:   prometheus.LinearBuckets(0, 0.003, 10),
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
