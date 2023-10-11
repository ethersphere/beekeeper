package smoke

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
	UploadDuration      prometheus.Histogram
	DownloadDuration    prometheus.Histogram
	UploadSize          prometheus.Gauge
}

func newMetrics(subsystem string) metrics {
	return metrics{
		BatchCreateAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "batch_create_attempts",
				Help:      "Number of batch create attempts.",
			},
		),
		BatchCreateErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "batch_create_errors",
				Help:      "Total errors encountered while creating batches.",
			},
		),
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
		DownloadMismatch: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_mismatch",
				Help:      "The total number of times uploaded data is different from downloaded data.",
			},
		),
		UploadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_upload_duration",
				Help:      "Data upload duration through the /bytes endpoint.",
			},
		),
		DownloadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_download_duration",
				Help:      "Data download duration through the /bytes endpoint.",
			},
		),
		UploadSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_size",
				Help:      "Amount of data uploaded per upload.",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}

func (c *LoadCheck) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
