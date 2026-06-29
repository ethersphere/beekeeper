package reserveradius

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	StorageRadius       *prometheus.GaugeVec
	ReserveSize         *prometheus.GaugeVec
	ReserveWithinRadius *prometheus.GaugeVec
	PullsyncRate        *prometheus.GaugeVec
	TimeToIncrease      prometheus.Gauge
	TimeToDecrease      prometheus.Gauge
	TimeToFullySynced   prometheus.Gauge
	RadiusTransitions   *prometheus.CounterVec
	RecoveryObserved    *prometheus.CounterVec
	Disruptions         *prometheus.CounterVec
	// redistribution-game liveness (observe mode), from /redistributionstate
	FullySynced        *prometheus.GaugeVec
	Frozen             *prometheus.GaugeVec
	RedistRound        *prometheus.GaugeVec
	LastSampleDuration *prometheus.GaugeVec
}

func newMetrics(subsystem string) metrics {
	return metrics{
		StorageRadius: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "storage_radius",
				Help:      "Storage radius reported by /status, per node.",
			},
			[]string{"node"},
		),
		ReserveSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "reserve_size",
				Help:      "Reserve size (chunks) reported by /status, per node.",
			},
			[]string{"node"},
		),
		PullsyncRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "pullsync_rate",
				Help:      "Pull-sync rate reported by /status, per node.",
			},
			[]string{"node"},
		),
		TimeToIncrease: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "time_to_increase_seconds",
				Help:      "Seconds to reach the target storage radius from the first upload.",
			},
		),
		TimeToDecrease: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "time_to_decrease_seconds",
				Help:      "Seconds from uploads-stopped to the first observed radius decrease.",
			},
		),
		RadiusTransitions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "radius_transitions_total",
				Help:      "Observed storage-radius transitions, by node and direction (up/down).",
			},
			[]string{"node", "direction"},
		),
		RecoveryObserved: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "recovery_observed_total",
				Help:      "Pull-sync recovery outcome after a radius decrease, by node and result (recovered/timeout).",
			},
			[]string{"node", "result"},
		),
		Disruptions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "disruption_total",
				Help:      "Neighbourhood disruptions applied, by mechanism (node-churn/batch-expiry).",
			},
			[]string{"mechanism"},
		),
		ReserveWithinRadius: nodeGauge(subsystem, "reserve_within_radius", "Reserve chunks within the storage radius, per node (completeness signal)."),
		TimeToFullySynced: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "time_to_fully_synced_seconds", Help: "Seconds from a radius decrease to the node reporting isFullySynced again.",
		}),
		FullySynced:        nodeGauge(subsystem, "fully_synced", "Redistribution isFullySynced per node (1=yes, 0=no) — can the node play the game."),
		Frozen:             nodeGauge(subsystem, "frozen", "Redistribution isFrozen per node (1=frozen) — a halt symptom (skips rounds)."),
		RedistRound:        nodeGauge(subsystem, "redistribution_round", "Current redistribution round per node (stuck round = halt symptom)."),
		LastSampleDuration: nodeGauge(subsystem, "last_sample_duration_seconds", "Last reserve-sample duration per node, from /redistributionstate."),
	}
}

// nodeGauge builds a per-node GaugeVec in the check namespace/subsystem.
func nodeGauge(subsystem, name, help string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: m.Namespace, Subsystem: subsystem, Name: name, Help: help},
		[]string{"node"},
	)
}

// Report implements the metrics.Reporter interface.
func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
