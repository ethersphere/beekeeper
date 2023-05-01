package debugapi

import (
	"context"
	"net/http"
)

// DbIndicesService represents Bee's DbIndices service
type DbIndicesService service

// DbIndices represents DbIndices's response
type DbIndices map[string]int

// GetDbIndices gets db indices
func (d *DbIndicesService) GetDbIndices(ctx context.Context) (DbIndices, error) {
	resp := make(DbIndices)
	err := d.client.requestJSON(ctx, http.MethodGet, "/dbindices", nil, &resp)
	return resp, err
}
