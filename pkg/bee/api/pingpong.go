package api

import (
	"context"
	"net/http"
)

// PingPongService ...
type PingPongService service

// Pong ...
type Pong struct {
	RTT string `json:"rtt"`
}

// Ping ...
func (p *PingPongService) Ping(ctx context.Context, overlayAddress string) (resp Pong, err error) {
	err = p.client.requestJSON(ctx, http.MethodPost, "/pingpong/"+overlayAddress, nil, &resp)
	return
}
