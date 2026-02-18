package api

import (
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

type UploadOptions struct {
	Act               bool
	Pin               bool
	Tag               uint64
	BatchID           string
	Direct            bool
	ActHistoryAddress swarm.Address
	RLevel            redundancy.Level

	// Dirs
	IndexDocument string
	ErrorDocument string
}

type DownloadOptions struct {
	Act                    *bool
	ActHistoryAddress      *swarm.Address
	ActPublicKey           *swarm.Address
	ActTimestamp           *uint64
	Cache                  *bool
	RLevel                 redundancy.Level
	RedundancyFallbackMode *bool
	OnlyRootChunk          *bool
}
