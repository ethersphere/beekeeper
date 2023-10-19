package longavailability

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	DownloadErrors   prometheus.Counter
	DownloadAttempts prometheus.Counter
	DownloadDuration prometheus.Histogram
	DownloadSize     prometheus.Gauge
}

func newMetrics(subsystem string) metrics {
	return metrics{
		DownloadAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_attempts",
				Help:      "Number of download attempts.",
			}),
		DownloadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_errors_count",
				Help:      "The total number of errors encountered before successful download.",
			}),
		DownloadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_download_duration",
				Help:      "Data download duration through the /bytes endpoint.",
			}),
		DownloadSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_size",
				Help:      "Amount of data downloaded per download.",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
