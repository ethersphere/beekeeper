package manifest

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents manifest options
type Options struct {
	FilesInCollection int
	MaxPathnameLength int32
	Seed              int64
	PostageAmount     int64
	PostageWait       time.Duration
	PostageDepth      uint64
}

var errManifest = errors.New("manifest data mismatch")

// Check executes manifest check
func Check(c *bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)

	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.FlattenOverlays(ctx)
	if err != nil {
		return err
	}

	files, err := generateFiles(rnd, o.FilesInCollection, o.MaxPathnameLength)
	if err != nil {
		return err
	}

	tarReader, err := tarFiles(files)
	if err != nil {
		return err
	}

	tarFile := bee.NewBufferFile("", tarReader)
	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := c.NodeNames()
	node := sortedNodes[0]

	client := clients[node]

	batchID, err := client.GetOrCreateBatch(ctx, o.PostageDepth, o.PostageWait)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", node, err)
	}
	fmt.Printf("node %s: batch id %s\n", node, batchID)

	if err := client.UploadCollection(ctx, &tarFile, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %d: %w", 0, err)
	}

	lastNode := sortedNodes[len(sortedNodes)-1]
	try := 0

DOWNLOAD:
	time.Sleep(5 * time.Second)
	try++
	if try > 5 {
		return errors.New("failed getting manifest files after too many retries")
	}

	for i, file := range files {
		node := clients[lastNode]

		size, hash, err := node.DownloadManifestFile(ctx, tarFile.Address(), file.Name())
		if err != nil {
			fmt.Printf("Node %s. Error retrieving file: %v\n", lastNode, err)
			goto DOWNLOAD
		}

		if !bytes.Equal(file.Hash(), hash) {
			fmt.Printf("Node %s. File %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s/%s\n", lastNode, i, file.Size(), size, overlays[lastNode].String(), tarFile.Address().String(), file.Name())
			return errManifest
		}

		fmt.Printf("Node %s. File %d retrieved successfully. Node: %s File: %s/%s\n", lastNode, i, overlays[lastNode].String(), tarFile.Address().String(), file.Name())
		try = 0 // reset the retry counter for the next file
	}

	return nil
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
			Mode: 0600,
			Size: file.Size(),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		b, err := ioutil.ReadAll(file.DataReader())
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
