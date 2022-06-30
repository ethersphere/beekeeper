package pushsync

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedCounter     *prometheus.CounterVec
	UploadTimeGauge     *prometheus.GaugeVec
	UploadTimeHistogram prometheus.Histogram
	SyncedCounter       *prometheus.CounterVec
	NotSyncedCounter    *prometheus.CounterVec
}

func newMetrics() metrics {
	subsystem := "check_pushsync"
	return metrics{
		UploadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_uploaded_count",
				Help:      "Number of uploaded chunks.",
			},
			[]string{"node"},
		),
		UploadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_upload_duration_seconds",
				Help:      "Chunk upload duration Gauge.",
			},
			[]string{"node", "chunk"},
		),
		UploadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_upload_seconds",
				Help:      "Chunk upload duration Histogram.",
			},
		),
		SyncedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_synced_count",
				Help:      "Number of chunks that has been synced with the closest node.",
			},
			[]string{"node"},
		),
		NotSyncedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_not_synced_count",
				Help:      "Number of chunks that has not been synced with the closest node.",
			},
			[]string{"node"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
