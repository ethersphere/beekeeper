package chunkstream

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	UploadedCounter       *prometheus.CounterVec
	UploadErrorCounter    *prometheus.CounterVec
	UploadTimeHistogram   prometheus.Histogram
	DownloadedCounter     *prometheus.CounterVec
	DownloadTimeHistogram prometheus.Histogram
	NotRetrievedCounter   *prometheus.CounterVec
	NotFoundCounter       *prometheus.CounterVec
}

func newMetrics() metrics {
	subsystem := "check_chunkstream"
	return metrics{
		UploadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_uploaded_count",
				Help:      "Number of chunks uploaded over the chunk stream.",
			},
			[]string{"node"},
		),
		UploadErrorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_upload_error_count",
				Help:      "Number of chunks that failed to upload over the chunk stream.",
			},
			[]string{"node"},
		),
		UploadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_upload_seconds",
				Help:      "Time to stream a full batch of chunks up.",
				Buckets:   prometheus.LinearBuckets(0, 0.5, 10),
			},
		),
		DownloadedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_downloaded_count",
				Help:      "Number of chunks retrieved over the chunk stream.",
			},
			[]string{"node"},
		),
		DownloadTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_download_seconds",
				Help:      "Time to stream a full batch of chunks back down.",
				Buckets:   prometheus.LinearBuckets(0, 0.5, 10),
			},
		),
		NotRetrievedCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_not_retrieved_count",
				Help:      "Number of chunks that were requested but not delivered correctly.",
			},
			[]string{"node"},
		),
		NotFoundCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_not_found_count",
				Help:      "Number of unknown chunks correctly reported as not found.",
			},
			[]string{"node"},
		),
	}
}

func (c *Check) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(c.metrics)
}
