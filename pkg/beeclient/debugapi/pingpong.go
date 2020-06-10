package debugapi

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

// Ping pings given node
func (p *PingPongService) Ping(ctx context.Context, a swarm.Address) (resp Pong, err error) {
	err = p.client.requestJSON(ctx, http.MethodPost, "/pingpong/"+a.String(), nil, &resp)
	return
}
