package longavailability

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	DownloadErrors         *prometheus.CounterVec
	DownloadAttempts       *prometheus.CounterVec
	Retrieved              *prometheus.CounterVec
	FailedDownloadAttempts *prometheus.CounterVec
	DownloadDuration       *prometheus.HistogramVec
	DownloadSize           *prometheus.GaugeVec
	DownloadStatus         *prometheus.GaugeVec
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
		Retrieved: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "retrieved",
				Help:      "Number of successful downloads.",
			},
			labels,
		),
		FailedDownloadAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "failed_download_attempts",
				Help:      "Number of failed download attempts.",
			},
			labels,
		),
		DownloadErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_errors_count",
				Help:      "The total number of errors encountered before successful download.",
			},
			labels,
		),
		DownloadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "d_download_duration_seconds",
				Help:      "Data download duration through the /bytes endpoint.",
			},
			labels,
		),
		DownloadSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "d_download_size_bytes",
				Help:      "Amount of data downloaded per download.",
			},
			labels,
		),
		DownloadStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "d_download_status",
				Help:      "Download status.",
			},
			labels,
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
