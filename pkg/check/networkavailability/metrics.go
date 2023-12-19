package networkavailability

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	BatchCreateErrors   prometheus.Counter
	BatchCreateAttempts prometheus.Counter
	UploadErrors        prometheus.Counter
	UploadAttempts      prometheus.Counter
	DownloadErrors      prometheus.Counter
	DownloadMismatch    prometheus.Counter
	DownloadAttempts    prometheus.Counter
	UploadDuration      *prometheus.HistogramVec
	DownloadDuration    *prometheus.HistogramVec
	UploadSize          prometheus.Gauge
}

const subsystem = "net_avail"

func newMetrics() metrics {
	return metrics{
		UploadAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_attempts",
				Help:      "Number of upload attempts.",
			},
		),
		DownloadAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_attempts",
				Help:      "Number of download attempts.",
			},
		),
		UploadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_errors_count",
				Help:      "The total number of errors encountered before successful upload.",
			},
		),
		DownloadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_errors_count",
				Help:      "The total number of errors encountered before successful download.",
			},
		),
		UploadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_upload_duration",
				Help:      "Data upload duration through the /bytes endpoint.",
			}, []string{"success"},
		),
		DownloadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_download_duration",
				Help:      "Data download duration through the /bytes endpoint.",
			}, []string{"success"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
