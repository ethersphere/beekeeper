package smoke

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	Iterations           prometheus.Counter
	FileUploadErrors     prometheus.Counter
	FileDownloadErrors   prometheus.Counter
	FileUploadDuration   prometheus.Histogram
	FileDownloadDuration prometheus.Histogram
}

func newMetrics() metrics {
	subsystem := "check_smoke"
	return metrics{
		Iterations: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "iterations",
				Help:      "The total number of the test iterations.",
			},
		),
		FileUploadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_errors_count",
				Help:      "The total number of errors encountered before successful upload.",
			},
		),
		FileDownloadErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_errors_count",
				Help:      "The total number of errors encountered before successful download.",
			},
		),
		FileUploadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_upload_duration",
				Help:      "Data upload duration through the /bytes endpoint.",
			},
		),
		FileDownloadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_download_duration",
				Help:      "Data download duration through the /bytes endpoint.",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
