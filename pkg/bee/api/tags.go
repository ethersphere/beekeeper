package api

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// TagsService represents Bee's Tag service
type TagsService service

type TagResponse struct {
	Total     int64     `json:"total"`
	Synced    int64     `json:"synced"`
	Uid       uint32    `json:"uid"`
	StartedAt time.Time `json:"startedAt"`
	Processed int64     `json:"processed"`
}

// CreateTag creates new tag
func (p *TagsService) CreateTag(ctx context.Context) (resp TagResponse, err error) {

	err = p.client.requestJSON(ctx, http.MethodPost, "/tags", nil, &resp)
	return
}

// GetTag gets a new tag
func (p *TagsService) GetTag(ctx context.Context, tagUID uint32) (resp TagResponse, err error) {

	tag := strconv.FormatUint(uint64(tagUID), 10)

	err = p.client.requestJSON(ctx, http.MethodGet, "/tags/"+tag, nil, &resp)

	return resp, err
}

func (p *TagsService) WaitSync(ctx context.Context, tagUID uint32) (err error) {

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

				if tr.Synced >= tr.Total {
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
