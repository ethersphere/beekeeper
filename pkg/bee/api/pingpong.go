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
func (p *PingPongService) Ping(ctx context.Context, overlayAddress string) (pong Pong, err error) {
	var r Pong
	err = p.client.request(ctx, http.MethodPost, "/pingpong/"+overlayAddress, nil, &r)
	return r, err
}
