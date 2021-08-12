package pushsync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

type metrics struct {
	uploadedChunks   *prometheus.CounterVec
	downloadedChunks *prometheus.CounterVec
	downloadCount    *prometheus.CounterVec
}

func newMetrics(runID string, pusher *push.Pusher) metrics {
	namespace := "beekeeper"
	subsystem := "simulation_pushsync"

	addCollector := func(c prometheus.Collector) {
		if pusher != nil {
			pusher.Collector(c)
		}
	}

	uploadedChunks := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"run": runID,
			},
			Name: "chunks_uploaded_count",
			Help: "Number of uploaded chunks.",
		},
		[]string{"node"},
	)
	addCollector(uploadedChunks)

	downloadedChunks := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"run": runID,
			},
			Name: "chunks_downloaded_count",
			Help: "Number of downloaded chunks.",
		},
		[]string{"node"},
	)
	addCollector(downloadedChunks)

	downloadCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"run": runID,
			},
			Name: "download_node_count",
			Help: "Number of nodes used for downloading",
		},
		[]string{"node"},
	)
	addCollector(downloadCount)

	if pusher != nil {
		pusher.Format(expfmt.FmtText)
	}

	return metrics{
		uploadedChunks:   uploadedChunks,
		downloadedChunks: downloadedChunks,
		downloadCount:    downloadCount,
	}
}
