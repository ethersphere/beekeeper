package soc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/cac"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/soc"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
)

// Options represents check options
type Options struct {
	NodeGroup string // TODO: support multi node group cluster
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		NodeGroup: "bee",
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	payload := []byte("Hello Swarm :)")

	ng := cluster.NodeGroup(o.NodeGroup)
	sortedNodes := ng.NodesSorted()

	privKey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return err
	}
	signer := crypto.NewDefaultSigner(privKey)

	ch, err := cac.New(payload)
	if err != nil {
		return err
	}

	idBytes, err := randomID()
	if err != nil {
		return err
	}
	sch, err := soc.New(idBytes, ch).Sign(signer)
	if err != nil {
		return err
	}

	chunkData := sch.Data()
	signatureBytes := chunkData[soc.IdSize : soc.IdSize+soc.SignatureSize]

	publicKey, err := signer.PublicKey()
	if err != nil {
		return err
	}

	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	nodeName := sortedNodes[0]
	node := ng.NodeClient(nodeName)

	owner := hex.EncodeToString(ownerBytes)
	id := hex.EncodeToString(idBytes)
	sig := hex.EncodeToString(signatureBytes)

	fmt.Printf("soc: submitting soc chunk %s to node %s\n", sch.Address().String(), nodeName)
	fmt.Printf("soc: owner %s\n", owner)
	fmt.Printf("soc: id %s\n", id)
	fmt.Printf("soc: sig %s\n", sig)

	ref, err := node.UploadSOC(ctx, owner, id, sig, ch.Data())
	if err != nil {
		return err
	}

	retrieved, err := node.DownloadChunk(ctx, ref, "")
	if err != nil {
		return err
	}

	if !bytes.Equal(retrieved, chunkData) {
		return errors.New("soc: retrieved chunk data does NOT match soc chunk")
	}

	return nil
}

func randomID() ([]byte, error) {
	key := make([]byte, 32)

	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}
