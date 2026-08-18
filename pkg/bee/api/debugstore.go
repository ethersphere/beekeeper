package api

import (
	"context"
	"net/http"
)

// DebugStoreService represents Bee's debug store service
type DebugStoreService service

// DebugStore represents DebugStore's response
type DebugStore struct {
	Upload     UploadStat     `json:"upload"`
	Pinning    PinningStat    `json:"pinning"`
	Cache      CacheStat      `json:"cache"`
	Reserve    ReserveStat    `json:"reserve"`
	ChunkStore ChunkStoreStat `json:"chunkStore"`
}

// UploadStat reports the upload store, which holds chunks the pusher has not yet
// delivered to the network. PendingUpload is that undelivered backlog.
type UploadStat struct {
	TotalUploaded int `json:"totalUploaded"`
	TotalSynced   int `json:"totalSynced"`
	PendingUpload int `json:"pendingUpload"`
}

type PinningStat struct {
	TotalCollections int `json:"totalCollections"`
	TotalChunks      int `json:"totalChunks"`
}

type CacheStat struct {
	Size     int `json:"size"`
	Capacity int `json:"capacity"`
}

type ReserveStat struct {
	SizeWithinRadius int      `json:"sizeWithinRadius"`
	TotalSize        int      `json:"totalSize"`
	Capacity         int      `json:"capacity"`
	LastBinIDs       []uint64 `json:"lastBinIDs"`
	Epoch            uint64   `json:"epoch"`
}

type ChunkStoreStat struct {
	TotalChunks    int `json:"totalChunks"`
	SharedSlots    int `json:"sharedSlots"`
	ReferenceCount int `json:"referenceCount"`
}

// GetDebugStore gets db indices
func (d *DebugStoreService) GetDebugStore(ctx context.Context) (DebugStore, error) {
	var resp DebugStore
	err := d.client.requestJSON(ctx, http.MethodGet, "/debugstore", nil, &resp)
	return resp, err
}