package pingpong

import "github.com/prometheus/client_golang/prometheus"

var (
	rttGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pingpong",
			Name:      "rtt_seconds",
			Help:      "Ping round-trip time Gauge",
		},
		[]string{"node", "peer"},
	)
	rttHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pingpong",
			Name:      "rtt_duration_seconds",
			Help:      "Ping round-trip time Histogram",
			Buckets:   prometheus.LinearBuckets(0, 0.003, 10),
		},
	)
)
