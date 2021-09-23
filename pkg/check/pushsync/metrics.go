package pushsync

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedCounter     *prometheus.CounterVec
	UploadTimeGauge     *prometheus.GaugeVec
	UploadTimeHistogram prometheus.Histogram
	SyncedCounter       *prometheus.CounterVec
	NotSyncedCounter    *prometheus.CounterVec

	CheckRun        *prometheus.CounterVec
	CheckFail       *prometheus.CounterVec
	CheckSuccess    *prometheus.CounterVec
	NodeSyncTime    *prometheus.HistogramVec
	RetrieveAttempt *prometheus.CounterVec
	RetrieveFail    *prometheus.CounterVec
	UploadSuccess   *prometheus.CounterVec
	RetrieveSuccess *prometheus.CounterVec
}

func newMetrics() metrics {
	subsystem := "check_pushsync"
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
			},
		),
		SyncedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_synced_count",
				Help:      "Number of chunks that has been synced with the closest node.",
			},
			[]string{"node"},
		),
		NotSyncedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_not_synced_count",
				Help:      "Number of chunks that has not been synced with the closest node.",
			},
			[]string{"node"},
		),

		CheckRun: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "check_run",
				Help:      "Times the check ran",
			},
			[]string{"node"},
		),
		CheckFail: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "check_fail",
				Help:      "Times the check failed",
			},
			[]string{"node"},
		),
		CheckSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "check_success",
				Help:      "Times the check succeeded.",
			},
			[]string{"node"},
		),

		NodeSyncTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "node_sync_time",
				Help:      "Time to availability of chunk on Nth node.",
			},
			[]string{"node", "index"},
		),
		RetrieveAttempt: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "retrieve_attempt",
				Help:      "Retrieval attempts.",
			},
			[]string{"node"},
		),
		RetrieveFail: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "retrieve_fail",
				Help:      "Retrieval failures.",
			},
			[]string{"node"},
		),
		RetrieveSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "retrieve_success",
				Help:      "Retrieval success.",
			},
			[]string{"node"},
		),
		UploadSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "upload_success",
				Help:      "Successful uploads.",
			},
			[]string{"node"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
