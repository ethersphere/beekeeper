package smoke

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	BatchCreateErrors   prometheus.Counter
	BatchCreateAttempts prometheus.Counter
	UploadErrors        *prometheus.CounterVec
	UploadAttempts      *prometheus.CounterVec
	UploadSuccess       *prometheus.CounterVec
	DownloadErrors      *prometheus.CounterVec
	DownloadMismatch    *prometheus.CounterVec
	DownloadAttempts    *prometheus.CounterVec
	DownloadSuccess     *prometheus.CounterVec
	UploadDuration      *prometheus.HistogramVec
	DownloadDuration    *prometheus.HistogramVec
	UploadThroughput    *prometheus.GaugeVec
	DownloadThroughput  *prometheus.GaugeVec
	UploadedBytes       *prometheus.CounterVec
	DownloadedBytes     *prometheus.CounterVec
}

const (
	labelSizeBytes       = "size_bytes"
	labelNodeName        = "node_name"
	labelRedundancyLevel = "redundancy_level"
)

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
		UploadAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_attempts",
				Help:      "Number of upload attempts.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_attempts",
				Help:      "Number of download attempts.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		UploadErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_errors_count",
				Help:      "The total number of errors encountered before successful upload.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_errors_count",
				Help:      "The total number of errors encountered before successful download.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadMismatch: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_mismatch",
				Help:      "The total number of times uploaded data is different from downloaded data.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
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
		UploadSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "upload_success",
				Help:      "Number of successful uploads.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		DownloadSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_success",
				Help:      "Number of successful downloads with matching data.",
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
