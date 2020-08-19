package debugapi

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// ChunksService represents Bee's debug Chunks service
type ChunksService service

// Remove removed the chunk from the node's local store
func (c *ChunksService) Remove(ctx context.Context, a swarm.Address) (err error) {
	err = c.client.request(ctx, http.MethodDelete, "/chunks/"+a.String(), nil, nil)
	return
}
