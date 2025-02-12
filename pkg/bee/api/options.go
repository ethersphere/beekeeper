package api

import "github.com/ethersphere/bee/v2/pkg/swarm"

type UploadOptions struct {
	Act               bool
	Pin               bool
	Tag               uint64
	BatchID           string
	Direct            bool
	ActHistoryAddress swarm.Address

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
	RedundancyFallbackMode *bool
	OnlyRootChunk          *bool
}
