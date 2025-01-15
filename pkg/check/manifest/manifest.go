package manifest

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

// Options represents check options
type Options struct {
	FilesInCollection int
	GasPrice          string
	MaxPathnameLength int32
	PostageAmount     int64
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
		PostageAmount:     1,
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

var errManifest = errors.New("manifest data mismatch")

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnd := random.PseudoGenerator(o.Seed)
	perm := rnd.Perm(cluster.Size())
	names := cluster.FullNodeNames()

	if len(names) < 2 {
		return fmt.Errorf("not enough nodes to run feed check")
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}
	upClient := clients[names[perm[0]]]
	downClient := clients[names[perm[1]]]

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
	batchID, err := upClient.GetOrCreateMutableBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", upClient.Name(), err)
	}
	c.logger.Infof("node %s: batch id %s", upClient.Name(), batchID)

	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %d: %w", 0, err)
	}

	for _, file := range files {
		if err := c.download(downClient, tarFile.Address(), &file, bee.File{}); err != nil {
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

	batchID, err := upClient.GetOrCreateMutableBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
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
	tarFile := bee.NewBufferFile("", tarReader)
	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return err
	}
	c.logger.Infof("collection uploaded: %s", tarFile.Address())
	time.Sleep(3 * time.Second)

	//push first version of website to the feed
	ref, err := upClient.UpdateFeedWithReference(ctx, signer, topic, 0, tarFile.Address(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	c.logger.Infof("feed updated: %s", ref.Reference)

	// download root (index.html) from the feed
	err = c.download(downClient, rootFeedRef.Reference, nil, files[0])
	if err != nil {
		return err
	}

	// update index.html file
	tmp, err := generateFilesWithPaths(rnd, []string{"index.html"}, int(o.MaxPathnameLength))
	if err != nil {
		return err
	}
	files[0] = tmp[0]
	tarReader, err = tarFiles(files)
	tarFile = bee.NewBufferFile("", tarReader)
	if err := upClient.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID, Direct: true}); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)

	// push 2nd version of website to the feed
	ref, err = upClient.UpdateFeedWithReference(ctx, signer, topic, 1, tarFile.Address(), api.UploadOptions{BatchID: batchID})
	if err != nil {
		return err
	}
	c.logger.Infof("feed updated: %s", ref.Reference)

	// download updated index.html from the feed
	err = c.download(downClient, rootFeedRef.Reference, nil, files[0])
	if err != nil {
		return err
	}

	// download other paths and compare
	for i := 0; i < len(files); i++ {
		err = c.download(downClient, tarFile.Address(), &files[i], files[0])
		if err != nil {
			return err
		}
	}
	return nil
}

// download retrieves a file from the given address using the specified client.
// If the file parameter is nil, it downloads the index file in the collection.
func (c *Check) download(client *bee.Client, address swarm.Address, file *bee.File, indexFile bee.File) error {
	fName := ""
	if file != nil {
		fName = file.Name()
	}
	c.logger.Infof("downloading file: %s/%s", address, fName)

	try := 0
DOWNLOAD:
	time.Sleep(5 * time.Second)
	try++
	if try > 5 {
		return fmt.Errorf("failed getting manifest files after too many retries")
	}
	_, hash, err := client.DownloadManifestFile(context.Background(), address, fName)
	if err != nil {
		c.logger.Infof("node %s. Error retrieving file: %s", client.Name(), err.Error())
		goto DOWNLOAD
	}

	if file != nil {
		if !bytes.Equal(file.Hash(), hash) {
			c.logger.Infof("node %s. File hash does not match", client.Name())
			return errManifest
		}
	} else {
		if !bytes.Equal(indexFile.Hash(), hash) {
			c.logger.Infof("node %s. Index hash does not match", client.Name())
			return errManifest
		}
	}
	c.logger.Infof("node %s. File retrieved successfully", client.Name())
	return nil
}

func generateFilesWithPaths(r *rand.Rand, paths []string, maxSize int) ([]bee.File, error) {
	files := make([]bee.File, len(paths))
	for i := 0; i < len(paths); i++ {
		path := paths[i]
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

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return &buf, nil
}
