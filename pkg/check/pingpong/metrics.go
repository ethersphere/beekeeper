package pingpong

import "github.com/prometheus/client_golang/prometheus"

var (
	rttGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pingpong",
			Name:      "rtt_gauge_seconds",
			Help:      "Round-trip time of a ping",
		},
		[]string{"node", "peer"},
	)
	rttHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pingpong",
			Name:      "rtt_duration_seconds",
			Help:      "Round-trip time of a ping",
			Buckets:   prometheus.LinearBuckets(0, 0.003, 10),
		},
	)
)
