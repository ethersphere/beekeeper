package pushsync

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedChunks   *prometheus.CounterVec
	DownloadedChunks *prometheus.CounterVec
	DownloadCount    *prometheus.CounterVec
}

func newMetrics(runID string) metrics {
	subsystem := "simulation_pushsync"

	return metrics{
		UploadedChunks: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_uploaded_count",
				Help: "Number of uploaded chunks.",
			},
			[]string{"node"},
		),
		DownloadedChunks: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "chunks_downloaded_count",
				Help: "Number of downloaded chunks.",
			},
			[]string{"node"},
		),
		DownloadCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				ConstLabels: prometheus.Labels{
					"run": runID,
				},
				Name: "download_node_count",
				Help: "Number of nodes used for downloading.",
			},
			[]string{"node"},
		),
	}
}

func (s *Simulation) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(s.metrics)
}
