package smoke

import (
	m "github.com/ethersphere/beekeeper/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	BatchCreateErrors      prometheus.Counter
	BatchCreateAttempts    prometheus.Counter
	UploadErrors           *prometheus.CounterVec
	UploadAttempts         *prometheus.CounterVec
	UploadSuccess          *prometheus.CounterVec
	DownloadErrors         *prometheus.CounterVec
	DownloadEOFErrors      *prometheus.CounterVec
	DownloadMismatch       *prometheus.CounterVec
	DownloadAttempts       *prometheus.CounterVec
	DownloadSuccess        *prometheus.CounterVec
	UploadDuration         *prometheus.HistogramVec
	DownloadDuration       *prometheus.HistogramVec
	UploadThroughput       *prometheus.GaugeVec
	DownloadThroughput     *prometheus.GaugeVec
	UploadedBytes          *prometheus.CounterVec
	DownloadedBytes        *prometheus.CounterVec
	NodeHealthVerdict      *prometheus.GaugeVec
	ClusterFullNodeCount   prometheus.Gauge
	ClusterLightNodeCount  prometheus.Gauge
	UnhealthyAbortsPreUp   prometheus.Counter
	UnhealthyAbortsPreDown prometheus.Counter
	// Chunk walk: per-chunk presence check across the full upload tree.
	ChunksChecked         prometheus.Counter
	ChunksMissingTotal    *prometheus.CounterVec // {position}
	ChunksMissingOutOfAOR *prometheus.CounterVec // {position} — bug 1 fingerprint (out-of-depth storing)
	ChunksMissingInAOR    *prometheus.CounterVec // {position} — bug 2/3 fingerprint (in-depth but not stored)
	ChunksPresentOutOfAOR *prometheus.CounterVec // {position} — bug 1 confirmed (chunk exists outside its AOR)
	FilesWithLoss         prometheus.Counter
}

const (
	labelSizeBytes       = "size_bytes"
	labelNodeName        = "node_name"
	labelRedundancyLevel = "redundancy_level"
	labelPhase           = "phase"
	labelPosition        = "position"
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
		DownloadEOFErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "download_eof_errors_count",
				Help:      "Download errors classified as unexpected EOF, which indicate the chunk is likely missing from the cluster.",
			},
			[]string{labelSizeBytes, labelNodeName, labelRedundancyLevel},
		),
		NodeHealthVerdict: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "node_health_verdict",
				Help:      "Topology health verdict for a node: 0=unknown, 1=unhealthy, 2=degraded, 3=healthy. Sampled per phase (pre_upload, pre_download, on_failure).",
			},
			[]string{labelNodeName, labelPhase},
		),
		ClusterFullNodeCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "cluster_full_node_count",
				Help:      "Number of full (non-bootnode) nodes in the cluster.",
			},
		),
		ClusterLightNodeCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "cluster_light_node_count",
				Help:      "Number of light nodes in the cluster.",
			},
		),
		UnhealthyAbortsPreUp: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "unhealthy_aborts_pre_upload",
				Help:      "Iterations aborted because the uploader was UNHEALTHY before upload.",
			},
		),
		UnhealthyAbortsPreDown: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "unhealthy_aborts_pre_download",
				Help:      "Iterations skipped because the downloader was UNHEALTHY before download.",
			},
		),
		ChunksChecked: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_checked_total",
				Help:      "Total chunks inspected by the on-failure chunk walk (denominator for chunks_missing_* rates).",
			},
		),
		ChunksMissingTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_missing_total",
				Help:      "Chunks not found on their closest full node (HEAD /chunks/{addr} returned 404). Labelled by tree position.",
			},
			[]string{labelPosition},
		),
		ChunksMissingOutOfAOR: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_missing_out_of_aor_total",
				Help:      "Missing chunks whose closest storer in the cluster still has PO(chunk, storer) < storageRadius. Indicates a cluster-coverage gap — the address falls outside every node's AOR. Common in small testnets.",
			},
			[]string{labelPosition},
		),
		ChunksMissingInAOR: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_missing_in_aor_total",
				Help:      "Missing chunks whose closest storer covers the address (PO >= storageRadius) but does not have the chunk. Bee#5400 bug-2/3 fingerprint: shallow receipt short-circuit or false ChunkSynced.",
			},
			[]string{labelPosition},
		),
		ChunksPresentOutOfAOR: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "chunks_present_out_of_aor_total",
				Help:      "Chunks held by a node whose AOR does not cover them (PO < storageRadius). Direct bee#5400 bug-1 confirmation: out-of-depth storing.",
			},
			[]string{labelPosition},
		),
		FilesWithLoss: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: m.Namespace,
				Subsystem: subsystem,
				Name:      "files_with_chunk_loss_total",
				Help:      "Files where the on-failure chunk walk found at least one missing chunk.",
			},
		),
	}
}

func (metrics *metrics) Report() []prometheus.Collector {
	return m.PrometheusCollectorsFromFields(*metrics)
}
