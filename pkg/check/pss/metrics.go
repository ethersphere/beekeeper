package pss

import "github.com/prometheus/client_golang/prometheus"

var (
	sendAndReceiveGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_pss",
			Name:      "pss_total_duration",
			Help:      "total duration between sending a message and receiving it on the other end.",
		},
		[]string{"nodeA", "nodeB"},
	)
)
