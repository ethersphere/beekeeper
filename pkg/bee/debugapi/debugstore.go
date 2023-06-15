package debugapi

import (
	"context"
	"net/http"
)

// DebugStoreService represents Bee's debug store service
type DebugStoreService service

// DebugStore represents DebugStore's response
type DebugStore map[string]int

// GetDebugStore gets db indices
func (d *DebugStoreService) GetDebugStore(ctx context.Context) (DebugStore, error) {
	resp := make(DebugStore)
	err := d.client.requestJSON(ctx, http.MethodGet, "/debugstore", nil, &resp)
	return resp, err
}
