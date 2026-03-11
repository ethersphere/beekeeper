package feed

import (
	beekeeperMetrics "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	FeedUpdateDurationSeconds    prometheus.Histogram
	FeedRetrievalDurationSeconds prometheus.Histogram
}

func newMetrics(subsystem string) metrics {
	return metrics{
		FeedUpdateDurationSeconds: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: beekeeperMetrics.Namespace,
				Subsystem: subsystem,
				Name:      "feed_update_duration_seconds",
				Help:      "Duration of each feed update (upload + UpdateFeed).",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
		),
		FeedRetrievalDurationSeconds: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: beekeeperMetrics.Namespace,
				Subsystem: subsystem,
				Name:      "feed_retrieval_duration_seconds",
				Help:      "Duration from FindFeedUpdate to DownloadFileBytes completion.",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
		),
	}
}

func (m *metrics) Report() []prometheus.Collector {
	return beekeeperMetrics.PrometheusCollectorsFromFields(*m)
}
