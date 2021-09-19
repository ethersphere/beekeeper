package retrieval

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedCounter       *prometheus.CounterVec
	UploadTimeGauge       *prometheus.GaugeVec
	UploadTimeHistogram   prometheus.Histogram
	DownloadedCounter     *prometheus.CounterVec
	DownloadTimeGauge     *prometheus.GaugeVec
	DownloadTimeHistogram prometheus.Histogram
	RetrievedCounter      *prometheus.CounterVec
	NotRetrievedCounter   *prometheus.CounterVec
}

func newMetrics() metrics {
	subsystem := "check_retrieval"
	return metrics{
		UploadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_uploaded_count",
				Help:      "Number of uploaded chunks.",
			},
			[]string{"node"},
		),
		UploadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_upload_duration_seconds",
				Help:      "Chunk upload duration Gauge.",
			},
			[]string{"node", "chunk"},
		),
		UploadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_upload_seconds",
				Help:      "Chunk upload duration Histogram.",
				Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
			},
		),
		DownloadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_downloaded_count",
				Help:      "Number of downloaded chunks.",
			},
			[]string{"node"},
		),
		DownloadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_download_duration_seconds",
				Help:      "Chunk download duration Gauge.",
			},
			[]string{"node", "chunk"},
		),
		DownloadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunk_download_seconds",
				Help:      "Chunk download duration Histogram.",
				Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
			},
		),
		RetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_retrieved_count",
				Help:      "Number of chunks that has been retrieved.",
			},
			[]string{"node"},
		),
		NotRetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_not_retrieved_count",
				Help:      "Number of chunks that has not been retrieved.",
			},
			[]string{"node"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
