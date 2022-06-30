package chunkrepair

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	RepairedCounter       *prometheus.CounterVec
	RepairedTimeGauge     *prometheus.GaugeVec
	RepairedTimeHistogram prometheus.Histogram
}

func newMetrics() metrics {
	subsystem := "check_chunkrepair"
	return metrics{
		RepairedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_repaired_count",
				Help:      "Number of chunks repaired.",
			},
			[]string{"node"},
		),
		RepairedTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_repaired_duration_seconds",
				Help:      "chunk repaired duration Gauge.",
			},
			[]string{"node", "file"},
		),
		RepairedTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_repaired_seconds",
				Help:      "Chunk repaired duration Histogram.",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
