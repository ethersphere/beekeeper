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
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents SOC check options
type Options struct {
	RequestTimeout time.Duration
	PostageAmount  int64
	PostageWait    time.Duration
	PostageDepth   uint64
}

// Check sends a SOC chunk and retrieves with the address.
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) error {

	payload := []byte("Hello Swarm :)")
	sortedNodes := c.NodeNames()

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

	ctx, cancel := context.WithTimeout(context.Background(), o.RequestTimeout)
	defer cancel()

	nodeName := sortedNodes[0]

	clients, err := c.NodesClients(ctx)
	if err != nil {
		return err
	}
	node := clients[nodeName]

	owner := hex.EncodeToString(ownerBytes)
	id := hex.EncodeToString(idBytes)
	sig := hex.EncodeToString(signatureBytes)

	batchID, err := node.GetOrCreateBatch(ctx, o.PostageDepth, o.PostageWait)
	if err != nil {
		return fmt.Errorf("node %s: batch id %w", nodeName, err)
	}
	fmt.Printf("node %s: batch id %s\n", nodeName, batchID)

	fmt.Printf("soc: submitting soc chunk %s to node %s\n", sch.Address().String(), nodeName)
	fmt.Printf("soc: owner %s\n", owner)
	fmt.Printf("soc: id %s\n", id)
	fmt.Printf("soc: sig %s\n", sig)

	ref, err := node.UploadSOC(ctx, owner, id, sig, ch.Data(), batchID)
	if err != nil {
		return err
	}

	fmt.Printf("soc: chunk uploaded to node %s\n", nodeName)

	retrieved, err := node.DownloadChunk(ctx, ref, "")
	if err != nil {
		return err
	}

	fmt.Printf("soc: chunk retrieved from node %s\n", nodeName)

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
