package retrieval

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedCounter       *prometheus.CounterVec
	NotUploadedCounter    *prometheus.CounterVec
	UploadTimeGauge       *prometheus.GaugeVec
	UploadTimeHistogram   prometheus.Histogram
	DownloadedCounter     *prometheus.CounterVec
	NotDownloadedCounter  *prometheus.CounterVec
	DownloadTimeGauge     *prometheus.GaugeVec
	DownloadTimeHistogram prometheus.Histogram
	RetrievedCounter      *prometheus.CounterVec
	NotRetrievedCounter   *prometheus.CounterVec
	SyncedCounter         *prometheus.CounterVec
	NotSyncedCounter      *prometheus.CounterVec
	SyncTagsTimeGauge     *prometheus.GaugeVec
	SyncTagsTimeHistogram prometheus.Histogram
}

func newMetrics(runID string) metrics {
	subsystem := "simulation_retrieval"

	return metrics{
		UploadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_uploaded_count",
				Help: "Number of uploaded chunks.",
			},
			[]string{"node"},
		),
		NotUploadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_not_uploaded_count",
				Help: "Number of not uploaded chunks.",
			},
			[]string{"node"},
		),
		UploadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunk_upload_duration_seconds",
				Help: "Chunk upload duration Gauge.",
			},
			[]string{"node", "chunk"},
		),
		UploadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name:    "chunk_upload_seconds",
				Help:    "Chunk upload duration Histogram.",
				Buckets: []float64{0.01, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
		),
		DownloadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_downloaded_count",
				Help: "Number of downloaded chunks.",
			},
			[]string{"node"},
		),
		NotDownloadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_not_downloaded_count",
				Help: "Number of chunks that has not been downloaded.",
			},
			[]string{"node"},
		),
		DownloadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunk_download_duration_seconds",
				Help: "Chunk download duration Gauge.",
			},
			[]string{"node", "chunk"},
		),
		DownloadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name:    "chunk_download_seconds",
				Help:    "Chunk download duration Histogram.",
				Buckets: []float64{0.01, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
		),
		RetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_retrieved_count",
				Help: "Number of chunks that has been retrieved.",
			},
			[]string{"node"},
		),
		NotRetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_not_retrieved_count",
				Help: "Number of chunks that has not been retrieved.",
			},
			[]string{"node"},
		),
		SyncedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "tags_synced_count",
				Help: "Number of synced tags.",
			},
			[]string{"node"},
		),
		NotSyncedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "tags_not_synced_count",
				Help: "Number of not synced tags.",
			},
			[]string{"node"},
		),
		SyncTagsTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "tags_sync_duration_seconds",
				Help: "Tags sync duration Gauge.",
			},
			[]string{"node", "chunk"},
		),
		SyncTagsTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name:    "tags_sync_seconds",
				Help:    "Tags sync duration Histogram.",
				Buckets: []float64{0.01, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
		),
	}
}

func (c *Simulation) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
