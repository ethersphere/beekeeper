package api

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// PingPongService represents Bee's PingPong service
type PingPongService service

// Pong represents Ping's response
type Pong struct {
	RTT string `json:"rtt"`
}

// Ping pings other node
func (p *PingPongService) Ping(ctx context.Context, overlay swarm.Address) (resp Pong, err error) {
	err = p.client.requestJSON(ctx, http.MethodPost, "/pingpong/"+overlay.String(), nil, &resp)
	return
}
