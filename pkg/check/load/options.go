package load

import "time"

type Options struct {
	ContentSize             int64
	RndSeed                 int64
	PostageTTL              time.Duration
	PostageDepth            uint64
	PostageLabel            string
	TxOnErrWait             time.Duration
	RxOnErrWait             time.Duration
	NodesSyncWait           time.Duration
	Duration                time.Duration
	UploaderCount           int
	UploadGroups            []string
	DownloaderCount         int
	DownloadGroups          []string
	MaxCommittedDepth       uint8
	CommittedDepthCheckWait time.Duration
	IterationWait           time.Duration
}

func NewDefaultOptions() Options {
	return Options{
		ContentSize:             5000000,
		RndSeed:                 time.Now().UnixNano(),
		PostageTTL:              24 * time.Hour,
		PostageDepth:            24,
		PostageLabel:            "test-label",
		TxOnErrWait:             10 * time.Second,
		RxOnErrWait:             10 * time.Second,
		NodesSyncWait:           time.Minute,
		Duration:                12 * time.Hour,
		MaxCommittedDepth:       2,
		CommittedDepthCheckWait: 5 * time.Minute,
		IterationWait:           5 * time.Minute,
	}
}
