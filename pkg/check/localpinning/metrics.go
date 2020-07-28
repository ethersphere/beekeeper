package localpinning

import "github.com/prometheus/client_golang/prometheus"

var (
	uploadedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_uploaded_count",
			Help:      "Number of uploaded files.",
		},
		[]string{"node"},
	)
	uploadTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "file_upload_duration_seconds",
			Help:      "File upload duration Gauge.",
		},
		[]string{"node", "file"},
	)
	uploadTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "file_upload_seconds",
			Help:      "File upload duration Histogram.",
			Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	downloadedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_downloaded_count",
			Help:      "Number of downloaded files.",
		},
		[]string{"node"},
	)
	downloadTimeGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "file_download_duration_seconds",
			Help:      "File download duration Gauge.",
		},
		[]string{"node", "file"},
	)
	downloadTimeHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "file_download_seconds",
			Help:      "File download duration Histogram.",
			Buckets:   prometheus.LinearBuckets(0, 0.1, 10),
		},
	)
	pinnedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_pinned_count",
			Help:      "Number of files that has been pinned.",
		},
		[]string{"node"},
	)
	notPinnedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_not_pinned_count",
			Help:      "Number of files that has not been pinned.",
		},
		[]string{"node"},
	)
	retrievedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_retrieved_count",
			Help:      "Number of files that has been retrieved.",
		},
		[]string{"node"},
	)
	notRetrievedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_not_retrieved_count",
			Help:      "Number of files that has not been retrieved.",
		},
		[]string{"node"},
	)
	unpinnedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_unpinned_count",
			Help:      "Number of files that has been unpinned.",
		},
		[]string{"node"},
	)
	notUnpinnedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "beekeeper",
			Subsystem: "check_localpinning",
			Name:      "files_not_unpinned_count",
			Help:      "Number of files that has not been unpinned.",
		},
		[]string{"node"},
	)
)
