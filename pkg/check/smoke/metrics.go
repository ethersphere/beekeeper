package smoke

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	Runs             prometheus.Counter
	ContentSize      prometheus.Gauge
	UploadDuration   prometheus.Gauge
	DownloadDuration prometheus.Gauge
}

func newMetrics() metrics {
	subsystem := "smoke"
	return metrics{
		Runs: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "runs",
				Help:      "Total number of runs.",
			},
		),
		ContentSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "filesize",
				Help:      "Size of the content being uploaded and downloaded in bytes.",
			},
		),
		UploadDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "upload_duration",
				Help:      "Duration in seconds for uploading the content.",
			},
		),
		DownloadDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "download_duration",
				Help:      "Duration in seconds for downloading the content.",
			},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
