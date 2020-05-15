package api

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// PingPongService ...
type PingPongService service

// Pong ...
type Pong struct {
	RTT string `json:"rtt"`
}

// Ping ...
func (p *PingPongService) Ping(ctx context.Context, overlay swarm.Address) (resp Pong, err error) {
	err = p.client.requestJSON(ctx, http.MethodPost, "/pingpong/"+overlay.String(), nil, &resp)
	return
}
