package manifest

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
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
	FilesInCollection int
	GasPrice          string
	MaxPathnameLength int32
	PostageTTL        time.Duration
	PostageDepth      uint64
	PostageLabel      string
	Seed              int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		FilesInCollection: 10,
		GasPrice:          "",
		MaxPathnameLength: 64,
		PostageTTL:        24 * time.Hour,
		PostageDepth:      16,
		PostageLabel:      "test-label",
		Seed:              0,
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

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
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

func (c *Check) checkWithoutSubDirs(ctx context.Context, rnd *rand.Rand, o Options, upClient *bee.Client, downClient *bee.Client) error {
	files, err := generateFiles(rnd, o.FilesInCollection, o.MaxPathnameLength)
	if err != nil {
		return err
	}

	tarReader, err := tarFiles(files)
	if err != nil {
		return err
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
			return err
		}
	}
	return nil
}

func (c *Check) checkWithSubDirs(ctx context.Context, rnd *rand.Rand, o Options, upClient *bee.Client, downClient *bee.Client) error {
	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return err
	}

	signer := crypto.NewDefaultSigner(privKey)
	topic, err := crypto.LegacyKeccak256([]byte("my-website"))
	if err != nil {
		return err
	}

	batchID, err := upClient.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", upClient.Name(), err)
	}
	c.logger.Infof("node %s: batch id %s", upClient.Name(), batchID)

	rootFeedRef, err := upClient.CreateRootFeedManifest(ctx, signer, topic, api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	c.logger.Infof("root feed reference: %s", rootFeedRef.Reference)
	time.Sleep(3 * time.Second)

	paths := []string{"index.html", "assets/styles/styles.css", "assets/styles/images/image.png", "error.html"}
	files, err := generateFilesWithPaths(rnd, paths, int(o.MaxPathnameLength))
	if err != nil {
		return err
	}

	tarReader, err := tarFiles(files)
	if err != nil {
		return err
	}
	tarFile := bee.NewBufferFile("", tarReader)
	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID, IndexDocument: "index.html"}); err != nil {
		return err
	}
	c.logger.Infof("collection uploaded: %s", tarFile.Address())

	time.Sleep(3 * time.Second)

	rChData, err := upClient.DownloadChunk(ctx, tarFile.Address(), "", nil)
	if err != nil {
		return err
	}
	// make chunk from byte array rChData
	rCh, err := cac.New(rChData)
	if err != nil {
		return err
	}
	c.logger.Infof("rChData downloaded: chunk data length %s", len(rChData))
	// push first version of website to the feed
	ref, err := upClient.UpdateFeedWithRootChunk(ctx, signer, topic, 0, rCh, api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	c.logger.Infof("feed updated: %s", ref.Reference)

	// download root (index.html) from the feed
	err = c.downloadAndVerify(ctx, downClient, rootFeedRef.Reference, nil, files[0])
	if err != nil {
		return err
	}

	// update  website files
	files, err = generateFilesWithPaths(rnd, paths, int(o.MaxPathnameLength))
	if err != nil {
		return err
	}

	tarReader, err = tarFiles(files)
	if err != nil {
		return err
	}
	tarFile = bee.NewBufferFile("", tarReader)
	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID, IndexDocument: "index.html"}); err != nil {
		return err
	}
	c.logger.Infof("collection uploaded: %s", tarFile.Address())
	time.Sleep(3 * time.Second)

	// Download Root Chunk of the new collection
	rChData, err = upClient.DownloadChunk(ctx, tarFile.Address(), "", nil)
	if err != nil {
		return err
	}
	rCh, err = cac.New(rChData)
	if err != nil {
		return err
	}
	c.logger.Infof("feed root chunk downloaded: %d bytes", len(rChData))
	// push 2nd version of website to the feed
	ref, err = upClient.UpdateFeedWithRootChunk(ctx, signer, topic, 1, rCh, api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	c.logger.Infof("feed updated: %s", ref.Reference)

	// download updated index.html from the feed
	err = c.downloadAndVerify(ctx, downClient, rootFeedRef.Reference, nil, files[0])
	if err != nil {
		return err
	}

	// download other paths and compare
	for i := 0; i < len(files); i++ {
		err = c.downloadAndVerify(ctx, downClient, tarFile.Address(), &files[i], files[0])
		if err != nil {
			return err
		}
	}
	return nil
}

// downloadAndVerify retrieves a file from the given address using the specified client.
// If the file parameter is nil, it downloads the index file in the collection.
// Then it verifies the hash of the downloaded file against the expected hash.
func (c *Check) downloadAndVerify(ctx context.Context, client *bee.Client, address swarm.Address, file *bee.File, indexFile bee.File) error {
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

func generateFilesWithPaths(r *rand.Rand, paths []string, maxSize int) ([]bee.File, error) {
	files := make([]bee.File, len(paths))
	for i, path := range paths {
		size := int64(r.Intn(maxSize)) + 1
		file := bee.NewRandomFile(r, path, size)
		err := file.CalculateHash()
		if err != nil {
			return nil, err
		}
		files[i] = file
	}
	return files, nil
}

func generateFiles(r *rand.Rand, filesCount int, maxPathnameLength int32) ([]bee.File, error) {
	files := make([]bee.File, filesCount)

	for i := 0; i < filesCount; i++ {
		pathnameLength := int64(r.Int31n(maxPathnameLength-1)) + 1 // ensure path with length of at least one

		b := make([]byte, pathnameLength)

		_, err := r.Read(b)
		if err != nil {
			return nil, err
		}

		pathname := hex.EncodeToString(b)

		file := bee.NewRandomFile(r, pathname, pathnameLength)

		err = file.CalculateHash()
		if err != nil {
			return nil, err
		}

		files[i] = file
	}

	return files, nil
}

// tarFiles receives an array of files and creates a new tar archive with those
// files as a collection.
func tarFiles(files []bee.File) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	defer tw.Close()

	for _, file := range files {
		// create tar header and write it
		hdr := &tar.Header{
			Name: file.Name(),
			Mode: 0o600,
			Size: file.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		b, err := io.ReadAll(file.DataReader())
		if err != nil {
			return nil, err
		}

		// write the file data to the tar
		if _, err := tw.Write(b); err != nil {
			return nil, err
		}
	}

	return &buf, nil
}
