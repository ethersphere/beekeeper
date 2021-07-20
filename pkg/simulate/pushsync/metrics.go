package pushsync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

type metrics struct {
	uploadedCounter   *prometheus.CounterVec
	downloadedCounter *prometheus.CounterVec
	downloadRetry     *prometheus.CounterVec
}

func newMetrics(runID string, pusher *push.Pusher) metrics {
	namespace := "beekeeper"
	subsystem := "simulation_pushsync"

	addCollector := func(c prometheus.Collector) {
		if pusher != nil {
			pusher.Collector(c)
		}
	}

	uploadedCounter := prometheus.NewCounterVec(
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
	addCollector(uploadedCounter)

	downloadedCounter := prometheus.NewCounterVec(
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
	addCollector(downloadedCounter)

	downloadRetry := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"run": runID,
			},
			Name: "download_retry_count",
			Help: "Number of download attempts",
		},
		[]string{"node"},
	)
	addCollector(downloadRetry)

	if pusher != nil {
		pusher.Format(expfmt.FmtText)
	}

	return metrics{
		uploadedCounter:   uploadedCounter,
		downloadedCounter: downloadedCounter,
		downloadRetry:     downloadRetry,
	}
}
