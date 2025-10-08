package feed

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	PostageTTL   time.Duration
	PostageDepth uint64
	PostageLabel string
	NUpdates     int
	RootRef      string
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		PostageTTL:   24 * time.Hour,
		PostageDepth: 17,
		PostageLabel: "test-label",
		NUpdates:     2,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		logger: logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.RootRef != "" {
		c.logger.Infof("running availability check")
		if err := c.checkAvailability(ctx, cluster, o); err != nil {
			return fmt.Errorf("availability check: %w", err)
		}
		return nil
	}

	c.logger.Infof("running feed check")
	if err := c.feedCheck(ctx, cluster, o); err != nil {
		return fmt.Errorf("feed check: %w", err)
	}

	return nil
}

func (c *Check) checkAvailability(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	ref, err := swarm.ParseHexAddress(o.RootRef)
	if err != nil {
		return fmt.Errorf("invalid root ref: %w", err)
	}

	clients, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("node clients: %w", err)
	}

	if len(clients) < 1 {
		return fmt.Errorf("availability check requires at least 1 full node")
	}

	_, _, err = clients[0].DownloadFile(ctx, ref, nil)
	if err != nil {
		return fmt.Errorf("download root feed: %w", err)
	}

	return nil
}

// feedCheck creates a root feed manifest, makes a series of updates to the feed
// and verifies that the updates are retrievable via another node.
func (c *Check) feedCheck(ctx context.Context, cluster orchestration.Cluster, o Options) error {
	clients, err := cluster.ShuffledFullNodeClients(ctx, random.PseudoGenerator(time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("node clients: %w", err)
	}

	if len(clients) < 2 {
		return fmt.Errorf("feed check requires at least 2 full nodes")
	}

	upClient := clients[0]
	downClient := clients[1]

	c.logger.Infof("upload client: %s", upClient.Name())

	batchID, err := upClient.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("get or create mutable batch: %w", err)
	}

	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return fmt.Errorf("generate private key: %w", err)
	}

	signer := crypto.NewDefaultSigner(privKey)
	topic, err := crypto.LegacyKeccak256([]byte("my-topic"))
	if err != nil {
		return fmt.Errorf("topic hash: %w", err)
	}

	// create root
	createManifestRes, err := upClient.CreateRootFeedManifest(ctx, signer, topic, api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}

	c.logger.Infof("node %s: manifest created", upClient.Name())
	c.logger.Infof("reference: %s", createManifestRes.Reference)
	c.logger.Infof("owner: %s", createManifestRes.Owner)
	c.logger.Infof("topic: %s", createManifestRes.Topic)

	// make updates
	for i := range o.NUpdates {
		time.Sleep(3 * time.Second)

		data := fmt.Sprintf("update-%d", i)
		fName := fmt.Sprintf("file-%d", i)
		file := bee.NewBufferFile(fName, bytes.NewBuffer([]byte(data)))
		err = upClient.UploadFile(context.Background(), &file, api.UploadOptions{
			BatchID: batchID,
			Direct:  true,
		})
		if err != nil {
			return fmt.Errorf("upload file `%s`: %w", fName, err)
		}

		// download root chunk of file
		rChData, err := upClient.DownloadChunk(ctx, file.Address(), "", nil)
		if err != nil {
			return fmt.Errorf("download root chunk: %w", err)
		}

		// make chunk from byte array rChData
		rCh, err := cac.NewWithDataSpan(rChData)
		if err != nil {
			return fmt.Errorf("create chunk: %w", err)
		}

		socRes, err := upClient.UpdateFeedWithRootChunk(ctx, signer, topic, uint64(i), rCh, api.UploadOptions{BatchID: batchID})
		if err != nil {
			return fmt.Errorf("update feed with root chunk: %w", err)
		}

		c.logger.Infof("node %s: feed updated", upClient.Name())
		c.logger.Infof("soc reference: %s", socRes.Reference)
		c.logger.Infof("wrapped reference: %s", file.Address())
	}
	time.Sleep(5 * time.Second)

	// fetch update
	c.logger.Infof("fetching feed update")
	c.logger.Infof("download client: %s", downClient.Name())
	update, err := downClient.FindFeedUpdate(ctx, signer, topic, nil)
	if err != nil {
		return fmt.Errorf("find feed update: %w", err)
	}

	c.logger.Infof("node %s: feed update found", downClient.Name())
	c.logger.Infof("index: %d", update.Index)
	c.logger.Infof("next index: %d", update.NextIndex)

	if update.NextIndex != uint64(o.NUpdates) {
		return fmt.Errorf("expected next index to be %d, got %d", o.NUpdates, update.NextIndex)
	}

	// fetch feed via bzz
	d, err := downClient.DownloadFileBytes(ctx, createManifestRes.Reference, nil)
	if err != nil {
		return fmt.Errorf("download root feed: %w", err)
	}

	lastUpdateData := fmt.Sprintf("update-%d", o.NUpdates-1)
	if string(d) != lastUpdateData {
		return fmt.Errorf("expected file content to be %s, got %s", lastUpdateData, string(d))
	}

	return nil
}
