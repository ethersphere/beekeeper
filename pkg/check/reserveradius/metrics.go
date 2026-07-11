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
	Dilutions           prometheus.Counter
}

func newMetrics(subsystem string) metrics {
	return metrics{
		StorageRadius:       nodeGauge(subsystem, "storage_radius", "Storage radius reported by /status, per node."),
		ReserveSize:         nodeGauge(subsystem, "reserve_size", "Reserve size (chunks) reported by /status, per node."),
		ReserveWithinRadius: nodeGauge(subsystem, "reserve_within_radius", "Reserve chunks within the storage radius, per node (completeness signal)."),
		PullsyncRate:        nodeGauge(subsystem, "pullsync_rate", "Pull-sync rate reported by /status, per node."),
		TimeToIncrease: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "time_to_increase_seconds", Help: "Seconds to reach the target storage radius from the first upload.",
		}),
		TimeToDecrease: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "time_to_decrease_seconds", Help: "Seconds from pull-sync going idle to the first observed radius decrease.",
		}),
		Dilutions: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "dilution_total", Help: "Postage-batch dilutions applied when forcing the decrease via expiry.",
		}),
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
