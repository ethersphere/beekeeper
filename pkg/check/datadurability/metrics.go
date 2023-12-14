package datadurability

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	ChunkDownloadAttempts prometheus.Counter
	ChunkDownloadErrors   prometheus.Counter
	ChunkDownloadDuration prometheus.Histogram
	ChunksCount           prometheus.Gauge
	FileDownloadAttempts  prometheus.Counter
	FileDownloadErrors    prometheus.Counter
	FileSize              prometheus.Counter
	FileDownloadDuration  prometheus.Histogram
}

func newMetrics(subsystem string) metrics {
	return metrics{
		ChunkDownloadAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_download_attempts",
				Help:      "Number of download attempts for the chunks.",
			},
		),
		ChunkDownloadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_download_errors",
				Help:      "Number of download errors for the chunks.",
			},
		),
		ChunkDownloadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_download_duration_seconds",
				Help:      "Chunk download duration through the /chunks endpoint.",
			},
		),
		ChunksCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_count",
				Help:      "The number of chunks in the check",
			},
		),
		FileDownloadAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_download_attempts",
				Help:      "Number of download attempts for the file.",
			},
		),
		FileDownloadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_download_errors",
				Help:      "Number of download errors for the file.",
			},
		),
		FileSize: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_size_bytes",
				Help:      "The size of the file downloaded (sum of chunk sizes)",
			},
		),
		FileDownloadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_download_duration_seconds",
				Help:      "File download duration",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
