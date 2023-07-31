package upload

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"golang.org/x/sync/errgroup"
)

var _ beekeeper.Action = (*Check)(nil)

type Options struct {
	Mode          string
	UploadCount   int
	PostageAmount int64
	PostageDepth  uint64
	Pin           bool
}

func NewDefaultOptions() Options {
	return Options{
		UploadCount:   1,
		PostageAmount: 1000000,
		PostageDepth:  20,
	}
}

type Check struct {
	logger logging.Logger
}

func NewCheck(logger logging.Logger) beekeeper.Action {
	return Check{
		logger: logger,
	}
}

func (c Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	switch o.Mode {
	case "direct":
		return c.directUploadCheck(ctx, cluster, o)
	case "deferred":
		return c.deferredUploadCheck(ctx, cluster, o)
	case "deferred_pin":
		o.Pin = true
		return c.deferredUploadCheck(ctx, cluster, o)
	default:
		return c.uploadCheck(ctx, cluster, o)
	}
}

func (c Check) uploadCheck(ctx context.Context, cluster orchestration.Cluster, opts Options) error {
	err := c.directUploadCheck(ctx, cluster, opts)
	if err != nil {
		return fmt.Errorf("direct upload: %w", err)
	}
	opts.Pin = true
	err = c.deferredUploadCheck(ctx, cluster, opts)
	if err != nil {
		return fmt.Errorf("deferred upload with pinning: %w", err)
	}
	return nil
}

func (c Check) directUploadCheck(ctx context.Context, cluster orchestration.Cluster, opts Options) error {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("get clients: %w", err)
	}

	nodeNames := cluster.NodeNames()
	uploadClient, downloadClient := clients[nodeNames[1]], clients[nodeNames[2]]

	for i := 0; i < opts.UploadCount; i++ {
		payload := make([]byte, bee.MaxChunkSize)
		_, err = rand.Read(payload)
		if err != nil {
			return fmt.Errorf("rand: %w", err)
		}

		batchID, err := uploadClient.GetOrCreateBatch(ctx, opts.PostageAmount, opts.PostageDepth, "", "upload-check")
		if err != nil {
			return fmt.Errorf("create batch: %w", err)
		}

		addr, err := uploadClient.UploadBytes(ctx, payload, api.UploadOptions{
			Direct:  true,
			BatchID: batchID,
		})
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}

		download, err := downloadClient.DownloadBytes(ctx, addr)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}

		if !bytes.Equal(payload, download) {
			return fmt.Errorf("expected payload and download to be equal")
		}
		c.logger.Infof("payload and download successful between node %s and %s", nodeNames[1], nodeNames[2])
	}

	return nil
}

func (c Check) deferredUploadCheck(ctx context.Context, cluster orchestration.Cluster, opts Options) error {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("get clients: %w", err)
	}

	nodeNames := cluster.NodeNames()
	uploadClient, downloadClient := clients[nodeNames[0]], clients[nodeNames[1]]

	tags := make([]api.TagResponse, opts.UploadCount)
	payloads := make([][]byte, opts.UploadCount)

	for i := 0; i < opts.UploadCount; i++ {
		payload := make([]byte, bee.MaxChunkSize)
		payloads[i] = payload

		_, err = rand.Read(payload)
		if err != nil {
			return fmt.Errorf("rand: %w", err)
		}

		tag, err := uploadClient.CreateTag(ctx)
		if err != nil {
			return fmt.Errorf("create tag: %w", err)
		}
		tags[i] = tag

		batchID, err := uploadClient.GetOrCreateBatch(ctx, opts.PostageAmount, opts.PostageDepth, "", "upload-check")
		if err != nil {
			return fmt.Errorf("create batch: %w", err)
		}

		_, err = uploadClient.UploadBytes(ctx, payload, api.UploadOptions{
			Tag:     tag.Uid,
			Direct:  false,
			Pin:     opts.Pin,
			BatchID: batchID,
		})
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, tag := range tags {
		g.Go(func() error {
			err = uploadClient.WaitSync(ctx, tag.Uid)
			if err != nil {
				return fmt.Errorf("sync tag %d: %w", tag.Uid, err)
			}
			return nil
		})
	}
	err = g.Wait()
	if err != nil {
		return fmt.Errorf("wait sync: %w", err)
	}

	for i, tag := range tags {
		download, err := downloadClient.DownloadBytes(ctx, tag.Address)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}

		if !bytes.Equal(payloads[i], download) {
			return fmt.Errorf("expected payload and download to be equal")
		}

		if opts.Pin {
			_, err = uploadClient.GetPinnedRootHash(ctx, tag.Address)
			if err != nil {
				return fmt.Errorf("check pin: %w", err)
			}
		}

		c.logger.Infof("upload and download successful between node %s and %s", nodeNames[0], nodeNames[1])
	}

	return nil
}
