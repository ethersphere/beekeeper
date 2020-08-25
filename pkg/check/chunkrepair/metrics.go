package chunkrepair

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	repairedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_chunkrepair",
			Name:      "chunk_repaired_count",
			Help:      "Number of chunks repaired.",
		},
		[]string{"node"},
	)
	repairedTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "chunk_repaired_duration_seconds",
			Help:      "chunk repaired duration Gauge.",
		},
		[]string{"node", "file"},
	)
	repairedTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "chunk_repaired_seconds",
			Help:      "Chunk repaired duration Histogram.",
			Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
)
