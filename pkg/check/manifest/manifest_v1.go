package manifest

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// compile check whether Check implements interface
var _ beekeeper.Action = (*CheckV1)(nil)

// Check instance.
type CheckV1 struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheckV1(logger logging.Logger) beekeeper.Action {
	return &CheckV1{
		logger: logger,
	}
}

func (c *CheckV1) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	clients, err := cluster.ShuffledFullNodeClients(ctx, rnd)
	if err != nil {
		return fmt.Errorf("node clients shuffle: %w", err)
	}

	if len(clients) < 2 {
		return fmt.Errorf("not enough nodes to run manifest check")
	}
	upClient := clients[0]
	downClient := clients[1]

	err = c.checkWithoutSubDirs(ctx, rnd, o, upClient, downClient)
	if err != nil {
		return fmt.Errorf("check without subdirs: %w", err)
	}

	err = c.checkWithSubDirs(ctx, rnd, o, upClient, downClient)
	if err != nil {
		return fmt.Errorf("check with subdirs: %w", err)
	}

	return nil
}

func (c *CheckV1) checkWithoutSubDirs(ctx context.Context, rnd *rand.Rand, o Options, upClient *bee.Client, downClient *bee.Client) error {
	files, err := generateFiles(rnd, o.FilesInCollection, o.MaxPathnameLength)
	if err != nil {
		return fmt.Errorf("generate files: %w", err)
	}

	tarReader, err := tarFiles(files)
	if err != nil {
		return fmt.Errorf("tar files: %w", err)
	}

	tarFile := bee.NewBufferFile("", tarReader)
	batchID, err := upClient.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", upClient.Name(), err)
	}
	c.logger.Infof("node %s: batch id %s", upClient.Name(), batchID)

	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %d: %w", 0, err)
	}

	for _, file := range files {
		if err := c.downloadAndVerify(ctx, downClient, tarFile.Address(), &file, bee.File{}); err != nil {
			return fmt.Errorf("download and verify: %w", err)
		}
	}
	return nil
}

func (c *CheckV1) checkWithSubDirs(ctx context.Context, rnd *rand.Rand, o Options, upClient *bee.Client, downClient *bee.Client) error {
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return fmt.Errorf("gen secp: %w", err)
	}

	signer := crypto.NewDefaultSigner(privKey)
	topic, err := crypto.LegacyKeccak256([]byte("my-website-v1"))
	if err != nil {
		return fmt.Errorf("keccak: %w", err)
	}

	batchID, err := upClient.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", upClient.Name(), err)
	}
	c.logger.Infof("node %s: batch id %s", upClient.Name(), batchID)

	rootFeedRef, err := upClient.CreateRootFeedManifest(ctx, signer, topic, api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("create root manifest: %w", err)
	}
	c.logger.Infof("root feed reference: %s", rootFeedRef.Reference)
	time.Sleep(3 * time.Second)

	paths := []string{"index.html", "assets/styles/styles.css", "assets/styles/images/image.png", "error.html"}
	files, err := generateFilesWithPaths(rnd, paths, int(o.MaxPathnameLength))
	if err != nil {
		return fmt.Errorf("generate files: %w", err)
	}

	tarReader, err := tarFiles(files)
	if err != nil {
		return fmt.Errorf("tar files: %w", err)
	}
	tarFile := bee.NewBufferFile("", tarReader)
	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID, IndexDocument: "index.html"}); err != nil {
		return fmt.Errorf("upload collection: %w", err)
	}
	c.logger.Infof("collection uploaded: %s", tarFile.Address())
	time.Sleep(3 * time.Second)

	// push first version of website to the feed
	ref, err := upClient.UpdateFeedWithReference(ctx, signer, topic, 0, tarFile.Address(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	c.logger.Infof("feed updated: %s", ref.Reference)

	// download root (index.html) from the feed
	err = c.downloadAndVerify(ctx, downClient, rootFeedRef.Reference, nil, files[0])
	if err != nil {
		return fmt.Errorf("download and verify: %w", err)
	}

	// update  website files
	files, err = generateFilesWithPaths(rnd, paths, int(o.MaxPathnameLength))
	if err != nil {
		return fmt.Errorf("generate files: %w", err)
	}

	tarReader, err = tarFiles(files)
	if err != nil {
		return fmt.Errorf("tar files: %w", err)
	}
	tarFile = bee.NewBufferFile("", tarReader)
	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID, IndexDocument: "index.html"}); err != nil {
		return fmt.Errorf("upload collection: %w", err)
	}
	c.logger.Infof("collection uploaded: %s", tarFile.Address())
	time.Sleep(3 * time.Second)

	// push 2nd version of website to the feed
	ref, err = upClient.UpdateFeedWithReference(ctx, signer, topic, 1, tarFile.Address(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return fmt.Errorf("update feed: %w", err)
	}
	c.logger.Infof("feed updated: %s", ref.Reference)

	// download updated index.html from the feed
	err = c.downloadAndVerify(ctx, downClient, rootFeedRef.Reference, nil, files[0])
	if err != nil {
		return fmt.Errorf("download and verify: %w", err)
	}

	// download other paths and compare
	for i := 0; i < len(files); i++ {
		err = c.downloadAndVerify(ctx, downClient, tarFile.Address(), &files[i], files[0])
		if err != nil {
			return fmt.Errorf("download and verify: %w", err)
		}
	}
	return nil
}

// downloadAndVerify retrieves a file from the given address using the specified client.
// If the file parameter is nil, it downloads the index file in the collection.
// Then it verifies the hash of the downloaded file against the expected hash.
func (c *CheckV1) downloadAndVerify(ctx context.Context, client *bee.Client, address swarm.Address, file *bee.File, indexFile bee.File) error {
	expectedHash := indexFile.Hash()
	fName := ""
	if file != nil {
		fName = file.Name()
		expectedHash = file.Hash()
	}
	c.logger.Infof("downloading file: %s/%s", address, fName)

	for i := 0; i < 10; i++ {
		select {
		case <-time.After(5 * time.Second):
			_, hash, err := client.DownloadManifestFile(ctx, address, fName)
			if err != nil {
				c.logger.Infof("node %s: error retrieving file: %s", client.Name(), err.Error())
				continue
			}

			c.logger.Infof("want hash: %s, got hash: %s", hex.EncodeToString(expectedHash), hex.EncodeToString(hash))
			if !bytes.Equal(expectedHash, hash) {
				c.logger.Infof("node %s: file hash does not match.", client.Name())
				continue
			}
			c.logger.Infof("node %s: file retrieved successfully", client.Name())
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("failed getting manifest file after too many retries")
}
