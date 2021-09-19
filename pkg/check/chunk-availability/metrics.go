package chunkavailability

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	Runs                prometheus.Counter
	Failures            prometheus.Counter
	UploadTimeHistogram prometheus.Histogram
	BroadcastPeersPeers prometheus.Counter
}

func newMetrics() metrics {
	subsystem := "chunk_availability"

	return metrics{
		Runs: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "runs",
				Help:      "How many checks were run.",
			},
		),
		Failures: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "failures",
				Help:      "How many checks failed.",
			},
		),
		UploadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_upload_seconds",
				Help:      "Chunk upload duration Histogram.",
			},
		),
	}
}

func (c *Check) Metrics() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
