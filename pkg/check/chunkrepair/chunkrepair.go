package chunkrepair

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents chunk repair check options
type Options struct {
	NumberOfChunksToRepair int
	Seed                   int64
}

// Check ...
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.NumberOfChunksToRepair)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(repairedCounter)
	pusher.Collector(repairedTimeGauge)
	pusher.Collector(repairedTimeHistogram)

	pusher.Format(expfmt.FmtText)

	for i := 0; i < o.NumberOfChunksToRepair; i++ {
		// Pick node A, B, C and a chunk which is closest to B
		nodeA, nodeB, nodeC, chunk, err := getNodes(ctx, c, rnds[i])
		if err != nil {
			return err
		}

		// upload the chunk in nodeA
		err = uploadAndPinChunkToNode(ctx, nodeA, chunk)
		if err != nil {
			return err
		}

		// check if the node is there in the local store of node B
		present, err := nodeB.HasChunk(ctx, chunk.Address())
		if err != nil {
			return err
		}
		if !present {
			return errors.New("nodeB could not retrieve the uploaded chunk")
		}

		// download the chunk from nodeC
		data, err := nodeC.DownloadChunk(ctx, chunk.Address())
		if err != nil {
			return err
		}
		if !bytes.Equal(data, chunk.Data()) {
			return errors.New("chunk downloaded in NodeC does not have proper data")
		}

		// delete the chunk from all nodes except nodeA
		addressA, err := nodeA.Overlay(ctx)
		if err != nil {
			return err
		}
		err = deleteChunkFromAllNodes(ctx, c, addressA, chunk)
		if err != nil {
			return err
		}

		// trigger downloading of the chunk from nodeC again (this time it should trigger chunk repair)
		data, err = nodeC.DownloadChunk(ctx, chunk.Address())
		if err != nil { // for status Accepted (209), a nil is returned
			return err
		}
		if data != nil { // if we had got 209, there should not be any data
			return err
		}
		time.Sleep(5 * time.Second) // give sometime so that the repair happens

		// download again to see if the chunk is repaired
		data, err = nodeC.DownloadChunk(ctx, chunk.Address())
		if err != nil { // this time it should succeed
			return err
		}
		if !bytes.Equal(data, chunk.Data()) {
			return errors.New("chunk downloaded in NodeC does not have proper data")
		}

		if pushMetrics {
			if err := pusher.Push(); err != nil {
				fmt.Printf("node %d: %s\n", i, err)
			}
		}
	}
	return nil
}

// getNodes get three nodes A, B, C and a chunk such that
// NodeA's and NodeC's first byte of the address does not match
// nodeB is the closest to the generated chunk in the cluster
func getNodes(ctx context.Context, c bee.Cluster, rnd *rand.Rand) (*bee.Node, *bee.Node, *bee.Node, *bee.Chunk, error) {
	var addressA *bee.Addresses
	var addressB *bee.Addresses
	var addressC *bee.Addresses
	var nodeA *bee.Node
	var nodeB *bee.Node
	var nodeC *bee.Node

	// get all the node's address of the cluster
	addresses, err := c.Addresses(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// find node A and C
	for _, a := range addresses {
		found := false
		for _, c := range addresses {
			// first bytes should not match
			if a.Overlay.Bytes()[0] != c.Overlay.Bytes()[0] {
				addressA = &a
				addressC = &c
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if addressA == nil || addressC == nil {
		return nil, nil, nil, nil, errors.New("could not find nodes with different first byte")
	}

	// find node B
	// generate a chunk and pick the closest address from all the available addresses
	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	found := false
	dist := big.NewInt(9999999999999999999) // some large distance to initialise
	for _, b := range addresses {
		// addressB should not be the same as addressA
		if bytes.Equal(b.Overlay.Bytes(), addressA.Overlay.Bytes()) {
			continue
		}
		// addressB should not be the same as addressC
		if bytes.Equal(b.Overlay.Bytes(), addressC.Overlay.Bytes()) {
			continue
		}
		currDist, err := Distance(b.Overlay.Bytes(), chunk.Address().Bytes())
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if currDist.Cmp(dist) == -1 {
			dist = currDist
			addressB = &b
			found = true
		}
	}
	if !found {
		return nil, nil, nil, nil, errors.New("could not find a node address closest to the generated chunk")
	}

	for _, node := range c.Nodes {
		addresses, err := node.Addresses(ctx)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if addresses.Overlay.Equal(addressA.Overlay) {
			nodeA = &node
		}
		if addresses.Overlay.Equal(addressB.Overlay) {
			nodeB = &node
		}
		if addresses.Overlay.Equal(addressC.Overlay) {
			nodeC = &node
		}
	}
	if nodeA == nil || nodeB == nil || nodeC == nil {
		return nil, nil, nil, nil, errors.New("could not find nodes from addresses")
	}

	return nodeA, nodeB, nodeC, &chunk, nil
}

func uploadAndPinChunkToNode(ctx context.Context, node *bee.Node, chunk *bee.Chunk) error {
	err := node.UploadChunk(ctx, chunk)
	if err != nil {
		return err
	}
	pinned, err := node.PinChunk(ctx, chunk.Address())
	if err != nil {
		return err
	}
	if !pinned {
		return errors.New("could not pin chunk in nodeA")
	}
	return nil
}

func deleteChunkFromAllNodes(ctx context.Context, c bee.Cluster, except swarm.Address, chunk *bee.Chunk) error {
	for _, node := range c.Nodes {
		addresses, err := node.Addresses(ctx)
		if err != nil {
			return err
		}

		// dont delete in the pinned node
		if addresses.Overlay.Equal(except) {
			continue
		}

		err = node.RemoveChunk(ctx, chunk)
		if err != nil {
			return err
		}
	}
	return nil
}

// Distance returns the distance between address x and address y as a (comparable) big integer using the distance metric defined in the swarm specification.
// Fails if not all addresses are of equal length.
func Distance(x, y []byte) (*big.Int, error) {
	distanceBytes, err := DistanceRaw(x, y)
	if err != nil {
		return nil, err
	}
	r := big.NewInt(0)
	r.SetBytes(distanceBytes)
	return r, nil
}

// DistanceRaw returns the distance between address x and address y in big-endian binary format using the distance metric defined in the swarm specfication.
// Fails if not all addresses are of equal length.
func DistanceRaw(x, y []byte) ([]byte, error) {
	if len(x) != len(y) {
		return nil, errors.New("address length must match")
	}
	c := make([]byte, len(x))
	for i, addr := range x {
		c[i] = addr ^ y[i]
	}
	return c, nil
}
