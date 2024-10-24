package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// TagsService represents Bee's Tag service
type TagsService service

type TagResponse struct {
	Split     uint64        `json:"split"`
	Seen      uint64        `json:"seen"`
	Stored    uint64        `json:"stored"`
	Sent      uint64        `json:"sent"`
	Synced    uint64        `json:"synced"`
	Uid       uint64        `json:"uid"`
	Address   swarm.Address `json:"address"`
	StartedAt time.Time     `json:"startedAt"`
}

// CreateTag creates new tag
func (p *TagsService) CreateTag(ctx context.Context) (resp TagResponse, err error) {
	err = p.client.requestJSON(ctx, http.MethodPost, "/tags", nil, &resp)
	return
}

// GetTag gets a new tag
func (p *TagsService) GetTag(ctx context.Context, tagUID uint64) (resp TagResponse, err error) {
	tag := strconv.FormatUint(tagUID, 10)

	err = p.client.requestJSON(ctx, http.MethodGet, "/tags/"+tag, nil, &resp)

	return resp, err
}

func (p *TagsService) WaitSync(ctx context.Context, tagUID uint64) (err error) {
	c := make(chan bool)
	defer close(c)

	e := make(chan error)
	defer close(e)

	go func(ctx context.Context, c chan bool, e chan error) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				tr, err := p.GetTag(ctx, tagUID)
				if err != nil {
					e <- err
					return
				}

				if tr.Split-tr.Seen == tr.Synced {
					c <- true
					return
				}

				time.Sleep(1000 * time.Millisecond)
			}
		}
	}(ctx, c, e)

	select {
	case <-c:
		return
	case err := <-e:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
