package reserveradius

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	StorageRadius  *prometheus.GaugeVec
	ReserveSize    *prometheus.GaugeVec
	PullsyncRate   *prometheus.GaugeVec
	TimeToIncrease prometheus.Gauge
	TimeToDecrease prometheus.Gauge
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
	}
}

// Report implements the metrics.Reporter interface.
func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
