package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// PSSService represents Bee's PSS service
type PSSService service

// Sends a PSS message to a recipienct with a specific topic
func (p *PSSService) SendMessage(ctx context.Context, nodeAddress swarm.Address, nodePublicKey string, topic string, data io.Reader) error {

	url := fmt.Sprintf("/%s/pss/send/%s/%s?recipient=%s", apiVersion, topic, nodeAddress.String()[:4], nodePublicKey)

	return p.client.request(ctx, http.MethodPost, url, data, nil)
}
