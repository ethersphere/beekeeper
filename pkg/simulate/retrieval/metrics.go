package retrieval

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

type metrics struct {
	uploadedCounter       *prometheus.CounterVec
	uploadTimeGauge       *prometheus.GaugeVec
	uploadTimeHistogram   prometheus.Histogram
	downloadedCounter     *prometheus.CounterVec
	downloadTimeGauge     *prometheus.GaugeVec
	downloadTimeHistogram prometheus.Histogram
	retrievedCounter      *prometheus.CounterVec
	notRetrievedCounter   *prometheus.CounterVec
}

func newMetrics(clusterName string, pusher *push.Pusher) metrics {
	namespace := "beekeeper"
	subsystem := "simulation_retrieval"

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
				"cluster": clusterName,
			},
			Name: "chunks_uploaded_count",
			Help: "Number of uploaded chunks.",
		},
		[]string{"node"},
	)
	addCollector(uploadedCounter)

	uploadTimeGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name: "chunk_upload_duration_seconds",
			Help: "Chunk upload duration Gauge.",
		},
		[]string{"node", "chunk"},
	)
	addCollector(uploadTimeGauge)

	uploadTimeHistogram := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name:    "chunk_upload_seconds",
			Help:    "Chunk upload duration Histogram.",
			Buckets: prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	addCollector(uploadTimeHistogram)

	downloadedCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name: "chunks_downloaded_count",
			Help: "Number of downloaded chunks.",
		},
		[]string{"node"},
	)
	addCollector(downloadedCounter)

	downloadTimeGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name: "chunk_download_duration_seconds",
			Help: "Chunk download duration Gauge.",
		},
		[]string{"node", "chunk"},
	)
	addCollector(downloadTimeGauge)

	downloadTimeHistogram := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name:    "chunk_download_seconds",
			Help:    "Chunk download duration Histogram.",
			Buckets: prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	addCollector(downloadTimeHistogram)

	retrievedCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name: "chunks_retrieved_count",
			Help: "Number of chunks that has been retrieved.",
		},
		[]string{"node"},
	)
	addCollector(retrievedCounter)

	notRetrievedCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			ConstLabels: prometheus.Labels{
				"cluster": clusterName,
			},
			Name: "chunks_not_retrieved_count",
			Help: "Number of chunks that has not been retrieved.",
		},
		[]string{"node"},
	)
	addCollector(notRetrievedCounter)

	if pusher != nil {
		pusher.Format(expfmt.FmtText)
	}

	return metrics{
		uploadedCounter:       uploadedCounter,
		uploadTimeGauge:       uploadTimeGauge,
		uploadTimeHistogram:   uploadTimeHistogram,
		downloadedCounter:     downloadedCounter,
		downloadTimeGauge:     downloadTimeGauge,
		downloadTimeHistogram: downloadTimeHistogram,
		retrievedCounter:      retrievedCounter,
		notRetrievedCounter:   notRetrievedCounter,
	}
}
