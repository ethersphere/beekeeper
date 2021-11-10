package smoke

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	FileUploadDuration   prometheus.Histogram
	FileDownloadDuration prometheus.Histogram
}

func newMetrics() metrics {
	subsystem := "check_smoke"
	return metrics{
		FileUploadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_upload_duration",
				Help:      "File upload duration through the /bzz endpoint.",
			},
		),
		FileDownloadDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_download_duration",
				Help:      "File download duration through the /bzz endpoint.",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
