package reserveradiusv2

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	StorageRadius       *prometheus.GaugeVec
	ReserveSize         *prometheus.GaugeVec
	ReserveWithinRadius *prometheus.GaugeVec
	ReserveFillPercent  *prometheus.GaugeVec
	PullsyncRate        *prometheus.GaugeVec
	RadiusEvents        *prometheus.CounterVec
	BatchesCreated      prometheus.Counter
	UploadedBytes       prometheus.Counter
	Dilutions           prometheus.Counter
	TimeToDecrease      prometheus.Gauge
}

func newMetrics(subsystem string) metrics {
	return metrics{
		StorageRadius:       nodeGauge(subsystem, "storage_radius", "Storage radius reported by /status, per node."),
		ReserveSize:         nodeGauge(subsystem, "reserve_size", "Reserve size (chunks) reported by /status, per node."),
		ReserveWithinRadius: nodeGauge(subsystem, "reserve_within_radius", "Reserve chunks within the storage radius, per node."),
		ReserveFillPercent:  nodeGauge(subsystem, "reserve_fill_percent", "Reserve fill as percent of configured capacity, per node."),
		PullsyncRate:        nodeGauge(subsystem, "pullsync_rate", "Pull-sync rate reported by /status, per node."),
		RadiusEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "radius_events_total", Help: "Storage radius changes observed, per node and direction (increase/decrease).",
		}, []string{"node", "direction"}),
		BatchesCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "batches_created_total", Help: "Postage batches created during the fill phase.",
		}),
		UploadedBytes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "uploaded_bytes_total", Help: "Bytes uploaded during the fill phase.",
		}),
		Dilutions: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "dilutions_total", Help: "Postage-batch dilutions applied during the decrease phase.",
		}),
		TimeToDecrease: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: m.Namespace, Subsystem: subsystem,
			Name: "time_to_decrease_seconds", Help: "Seconds from the start of the decrease phase to the first observed radius decrease.",
		}),
	}
}

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
