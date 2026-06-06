package smoke

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	BatchCreate        *prometheus.CounterVec
	Upload             *prometheus.CounterVec
	Download           *prometheus.CounterVec
	UploadDuration     *prometheus.HistogramVec
	DownloadDuration   *prometheus.HistogramVec
	UploadThroughput   *prometheus.GaugeVec
	DownloadThroughput *prometheus.GaugeVec
	UploadedBytes      *prometheus.CounterVec
	DownloadedBytes    *prometheus.CounterVec
}

const (
	labelSizeBytes       = "size_bytes"
	labelNodeName        = "node_name"
	labelRedundancyLevel = "redundancy_level"
	labelResult          = "result"
)

func newMetrics(subsystem string) metrics {
	return metrics{
		BatchCreate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "batch_total",
				Help:      "Number of batch create attempts by result.",
			},
			[]string{labelResult},
		),
		Upload: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_total",
				Help:      "Number of upload attempts by result.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel, labelResult},
		),
		Download: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_total",
				Help:      "Number of download attempts by result.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel, labelResult},
		),
		UploadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_upload_duration",
				Help:      "Data upload duration through the /bytes endpoint.",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 600, 1200, 1800, 3600},
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "data_download_duration",
				Help:      "Data download duration through the /bytes endpoint.",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 50, 100, 250, 600, 1200, 1800, 3600},
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		UploadThroughput: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_throughput_bytes_per_second",
				Help:      "Upload throughput in bytes per second.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadThroughput: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_throughput_bytes_per_second",
				Help:      "Download throughput in bytes per second.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		UploadedBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "uploaded_bytes_total",
				Help:      "Total bytes successfully uploaded.",
			},
			[]string{labelNodeName, labelRedundancyLevel},
		),
		DownloadedBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "downloaded_bytes_total",
				Help:      "Total bytes successfully downloaded.",
			},
			[]string{labelNodeName, labelRedundancyLevel},
		),
	}
}

func (metrics *metrics) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(*metrics)
}
