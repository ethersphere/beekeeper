package fileretrieval

import (
	m "github.com/ethersphere/bee/pkg/metrics"
	mm "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedCounter       *prometheus.CounterVec
	UploadTimeGauge       *prometheus.GaugeVec
	UploadTimeHistogram   prometheus.Histogram
	DownloadedCounter     *prometheus.CounterVec
	DownloadTimeGauge     *prometheus.GaugeVec
	DownloadTimeHistogram prometheus.Histogram
	RetrievedCounter      *prometheus.CounterVec
	NotRetrievedCounter   *prometheus.CounterVec
}

func newMetrics() metrics {
	subsystem := "check_fileretrieval"
	return metrics{
		UploadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "files_uploaded_count",
				Help:      "Number of uploaded files.",
			},
			[]string{"node"},
		),
		UploadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "file_upload_duration_seconds",
				Help:      "File upload duration Gauge.",
			},
			[]string{"node", "file"},
		),
		UploadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "file_upload_seconds",
				Help:      "File upload duration Histogram.",
				Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
			},
		),
		DownloadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "files_downloaded_count",
				Help:      "Number of downloaded files.",
			},
			[]string{"node"},
		),
		DownloadTimeGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "file_download_duration_seconds",
				Help:      "File download duration Gauge.",
			},
			[]string{"node", "file"},
		),
		DownloadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "file_download_seconds",
				Help:      "File download duration Histogram.",
				Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
			},
		),
		RetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "files_retrieved_count",
				Help:      "Number of files that has been retrieved.",
			},
			[]string{"node"},
		),
		NotRetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: mm.Namespace,
				Subsystem: subsystem,
				Name:      "files_not_retrieved_count",
				Help:      "Number of files that has not been retrieved.",
			},
			[]string{"node"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
