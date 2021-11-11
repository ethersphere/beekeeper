package smoke

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	Iterations           prometheus.Counter
	FileUploadDuration   prometheus.Gauge
	FileDownloadDuration prometheus.Gauge
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
		FileUploadDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "file_upload_duration",
				Help:      "File upload duration through the /bzz endpoint.",
			},
		),
		FileDownloadDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
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
