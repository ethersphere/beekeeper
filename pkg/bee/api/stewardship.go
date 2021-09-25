package api

import (
	"context"
	"net/http"

	"github.com/ethersphere/bee/pkg/swarm"
)

// StewardshipService represents Bee's Stewardship service.
type StewardshipService service

// stewardshipBasePath is the stewardship API base path for http requests.
const stewardshipBasePath = "/stewardship"

func stewardshipPath(path string) string { return stewardshipBasePath + "/" + path }

// IsRetrievable checks whether the content on the given address is retrievable.
func (ss *StewardshipService) IsRetrievable(ctx context.Context, ref swarm.Address) (bool, error) {
	res := struct {
		IsRetrievable bool `json:"isRetrievable"`
	}{}
	err := ss.client.requestJSON(ctx, http.MethodGet, stewardshipPath(ref.String()), nil, &res)
	if err != nil {
		return false, err
	}
	return res.IsRetrievable, nil
}

// Reupload re-uploads root hash and all of its underlying associated chunks to
// the network.
func (ss *StewardshipService) Reupload(ctx context.Context, ref swarm.Address) error {
	res := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}
	return ss.client.requestJSON(ctx, http.MethodPut, stewardshipPath(ref.String()), nil, &res)
}
