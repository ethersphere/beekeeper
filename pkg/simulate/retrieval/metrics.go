package retrieval

import "github.com/prometheus/client_golang/prometheus"

var (
	uploadedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunks_uploaded_count",
			Help:      "Number of uploaded chunks.",
		},
		[]string{"node"},
	)
	uploadTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunk_upload_duration_seconds",
			Help:      "Chunk upload duration Gauge.",
		},
		[]string{"node", "chunk"},
	)
	uploadTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunk_upload_seconds",
			Help:      "Chunk upload duration Histogram.",
			Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	downloadedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunks_downloaded_count",
			Help:      "Number of downloaded chunks.",
		},
		[]string{"node"},
	)
	downloadTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunk_download_duration_seconds",
			Help:      "Chunk download duration Gauge.",
		},
		[]string{"node", "chunk"},
	)
	downloadTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunk_download_seconds",
			Help:      "Chunk download duration Histogram.",
			Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	retrievedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunks_retrieved_count",
			Help:      "Number of chunks that has been retrieved.",
		},
		[]string{"node"},
	)
	notRetrievedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "simulation_retrieval",
			Name:      "chunks_not_retrieved_count",
			Help:      "Number of chunks that has not been retrieved.",
		},
		[]string{"node"},
	)
)
