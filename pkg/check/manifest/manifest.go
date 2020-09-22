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
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents manifest options
type Options struct {
	FilesInCollection int
	MaxPathnameLength int32
	Seed              int64
}

var errManifest = errors.New("manifest data mismatch")

func Check(c bee.Cluster, o Options) error {
	ctx := context.Background()
	rnd := random.PseudoGenerator(o.Seed)

	fmt.Printf("Seed: %d\n", o.Seed)

	overlays, err := c.Overlays(ctx)
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

	if err := c.Nodes[0].UploadCollection(ctx, &tarFile); err != nil {
		return fmt.Errorf("node %d: %w", 0, err)
	}

	lastNodeIndex := c.Size() - 1
	try := 0

DOWNLOAD:
	time.Sleep(5 * time.Second)
	try++
	if try > 5 {
		return errors.New("failed getting manifest files after too many retries")
	}

	for i, file := range files {
		size, hash, err := c.Nodes[lastNodeIndex].DownloadManifestFile(ctx, tarFile.Address(), file.Name())
		if err != nil {
			fmt.Printf("node %d: error retrieving file: %v", lastNodeIndex, err)
			goto DOWNLOAD
		}

		if !bytes.Equal(file.Hash(), hash) {
			fmt.Printf("Node %d. File %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s File: %s/%s\n", lastNodeIndex, i, file.Size(), size, overlays[lastNodeIndex].String(), tarFile.Address().String(), file.Name())
			return errManifest
		}

		fmt.Printf("Node %d. File %d retrieved successfully. Node: %s File: %s/%s\n", lastNodeIndex, i, overlays[lastNodeIndex].String(), tarFile.Address().String(), file.Name())
	}

	return nil
}

func generateFiles(r *rand.Rand, filesCount int, maxPathnameLength int32) ([]bee.File, error) {
	files := make([]bee.File, filesCount)

	for i := 0; i < filesCount; i++ {
		pathnameLength := int64(r.Int31n(maxPathnameLength))

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
