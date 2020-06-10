package pushsync

import "github.com/prometheus/client_golang/prometheus"

var (
	uploadedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pushsync",
			Name:      "chunks_uploaded",
			Help:      "Number of uploaded chunks.",
		},
		[]string{"node"},
	)
	uploadTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pushsync",
			Name:      "chunk_upload_seconds",
			Help:      "Chunk upload duration Gauge.",
		},
		[]string{"node", "chunk"},
	)
	uploadTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pushsync",
			Name:      "chunk_upload_duration_seconds",
			Help:      "Chunk upload duration Histogram.",
			Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	syncedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pushsync",
			Name:      "chunks_synced",
			Help:      "Number of chunks that has been synced with the closest node.",
		},
		[]string{"node"},
	)
	notSyncedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pushsync",
			Name:      "chunks_not_synced",
			Help:      "Number of chunks that has not been synced with the closest node.",
		},
		[]string{"node"},
	)
)
