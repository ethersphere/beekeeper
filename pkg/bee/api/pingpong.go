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
func (p *PingPongService) Ping(ctx context.Context, overylay swarm.Address) (resp Pong, err error) {
	err = p.client.requestJSON(ctx, http.MethodPost, "/pingpong/"+overylay.String(), nil, &resp)
	return
}
