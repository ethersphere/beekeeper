package topohealth

import (
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/sirupsen/logrus"
)

// Phase tags when in a check's lifecycle a probe was taken.
type Phase string

const (
	PhasePreUpload   Phase = "pre_upload"
	PhasePreDownload Phase = "pre_download"
	PhaseOnFailure   Phase = "on_failure"
)

// LogVerdict emits a structured log line for a per-node verdict. All probe
// data is attached as logrus fields; the message is a stable event name that
// CI log greps can pin on.
func LogVerdict(logger logging.Logger, phase Phase, v Verdict) {
	logger.WithFields(logrus.Fields{
		"event":             "topohealth.verdict",
		"phase":             string(phase),
		"node":              v.Node,
		"status":            v.Status.String(),
		"overlay":           v.Overlay.String(),
		"depth":             v.Raw.Depth,
		"connected":         v.Raw.Connected,
		"population":        v.Raw.Population,
		"empty_below_depth": v.Raw.EmptyBinsBelowDepth,
		"dial_ratio":        v.Raw.DialRatio,
		"reachability":      v.Raw.Reachability,
		"warming_up":        v.Raw.IsWarmingUp,
		"reserve_size":      v.Raw.ReserveSize,
		"storage_radius":    v.Raw.StorageRadius,
		"committed_depth":   v.Raw.CommittedDepth,
		"failure_reasons":   v.FailureReasons,
	}).Info("topohealth.verdict")
}

// LogChunkCheck emits one structured line per missing or out-of-AOR chunk
// from a WalkChunks result.
func LogChunkCheck(logger logging.Logger, kind, chunkAddr string, c ChunkCheck) {
	logger.WithFields(logrus.Fields{
		"event":          "topohealth.chunk_check",
		"kind":           kind,
		"upload_root":    chunkAddr,
		"chunk_address":  c.Address.String(),
		"position":       string(c.Position),
		"storer":         c.StorerName,
		"storer_overlay": c.StorerOverlay.String(),
		"proximity":      c.Proximity,
		"storage_radius": c.StorageRadius,
		"out_of_aor":     c.OutOfAOR,
		"present":        c.Present,
		"head_error":     c.Error,
	}).Info("topohealth.chunk_check")
}

// LogStorerResult emits one structured line per intended-storer probe,
// including the HEAD /chunks/{addr} ground-truth.
func LogStorerResult(logger logging.Logger, chunkAddr, phase string, idx int, r StorerResult) {
	logger.WithFields(logrus.Fields{
		"event":           "topohealth.storer",
		"phase":           phase,
		"storer_rank":     idx,
		"chunk_addr":      chunkAddr,
		"node":            r.Verdict.Node,
		"status":          r.Verdict.Status.String(),
		"has_chunk":       r.HasChunk,
		"has_error":       r.HasError,
		"failure_reasons": r.Verdict.FailureReasons,
	}).Info("topohealth.storer")
}
