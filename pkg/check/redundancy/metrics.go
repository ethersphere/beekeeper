package redundancy

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	DownloadAttempts *prometheus.CounterVec
	DownloadErrors   *prometheus.CounterVec
	DownloadDuration *prometheus.HistogramVec
}

func newMetrics(subsystem string, labels []string) metrics {
	return metrics{
		DownloadAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_attempts",
				Help:      "Number of download attempts.",
			},
			labels,
		),
		DownloadErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_errors",
				Help:      "Number of download errors",
			},
			labels,
		),
		DownloadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_duration_seconds",
				Help:      "Download duration",
			},
			labels,
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
