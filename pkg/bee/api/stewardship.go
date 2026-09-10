package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// StewardshipService represents Bee's Stewardship service.
type StewardshipService service

// stewardshipBasePath is the stewardship API base path for http requests.
const stewardshipBasePath = "/stewardship"

func stewardshipPath(path string) string { return stewardshipBasePath + "/" + path }

// StewardshipOptions holds the headers the stewardship endpoints accept.
// A nil RLevel omits the Swarm-Redundancy-Level header, which makes the node
// fall back to redundancy.DefaultUploadLevel.
type StewardshipOptions struct {
	BatchID string
	RLevel  *redundancy.Level
}

func (o StewardshipOptions) header() http.Header {
	h := http.Header{}
	if o.BatchID != "" {
		h.Add(postageStampBatchHeader, o.BatchID)
	}
	if o.RLevel != nil {
		h.Add(swarmRedundancyLevelHeader, strconv.Itoa(int(*o.RLevel)))
	}
	return h
}

// IsRetrievable checks whether the content on the given address is retrievable.
func (ss *StewardshipService) IsRetrievable(ctx context.Context, ref swarm.Address, o StewardshipOptions) (bool, error) {
	res := struct {
		IsRetrievable bool `json:"isRetrievable"`
	}{}
	if err := ss.client.requestWithHeader(ctx, http.MethodGet, stewardshipPath(ref.String()), o.header(), nil, &res); err != nil {
		return false, err
	}
	return res.IsRetrievable, nil
}

// Reupload re-uploads root hash and all of its underlying associated chunks to
// the network. The postage batch is required by the node.
func (ss *StewardshipService) Reupload(ctx context.Context, ref swarm.Address, o StewardshipOptions) error {
	res := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}
	return ss.client.requestWithHeader(ctx, http.MethodPut, stewardshipPath(ref.String()), o.header(), nil, &res)
}
