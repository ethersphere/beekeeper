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
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents check options
type Options struct {
	MetricsPusher          *push.Pusher
	NodeGroup              string // TODO: support multi node group cluster
	NumberOfChunksToRepair int
	PostageAmount          int64
	PostageWait            time.Duration
	Seed                   int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		MetricsPusher:          nil,
		NodeGroup:              "bee",
		NumberOfChunksToRepair: 1,
		PostageAmount:          1,
		PostageWait:            5 * time.Second,
		Seed:                   0,
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
	fmt.Println("running chunk repair")
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnds := random.PseudoGenerators(o.Seed, o.NumberOfChunksToRepair)
	fmt.Printf("Seed: %d\n", o.Seed)

	if o.MetricsPusher != nil {
		o.MetricsPusher.Collector(repairedCounter)
		o.MetricsPusher.Collector(repairedTimeGauge)
		o.MetricsPusher.Collector(repairedTimeHistogram)
		o.MetricsPusher.Format(expfmt.FmtText)
	}

	ng := cluster.NodeGroup(o.NodeGroup)
	for i := 0; i < o.NumberOfChunksToRepair; i++ {
		// Pick node A, B, C and a chunk which is closest to B
		nodeA, nodeB, nodeC, chunk, err := getNodes(ctx, ng, rnds[i])
		if err != nil {
			return err
		}
		addressA, err := nodeA.Overlay(ctx)
		if err != nil {
			return err
		}

		batchID, err := nodeA.CreatePostageBatch(ctx, o.PostageAmount, bee.MinimumBatchDepth, "test-label")
		if err != nil {
			return fmt.Errorf("created batched id %w", err)
		}

		fmt.Printf("created batched id %s", batchID)

		time.Sleep(o.PostageWait)

		// upload the chunk in nodeA
		ref, err := nodeA.UploadChunk(ctx, chunk.Data(), api.UploadOptions{BatchID: batchID})
		if err != nil {
			return err
		}

		count := 0
		for {
			if count > maxIterations {
				return fmt.Errorf("could not get chunk even after several attempts")
			}

			// check if the node is there in the local store of node B
			// this does a get chunk instead of Has chunk, so the following
			// call just checks if the chunk is accessible from nodeB
			present, err := nodeB.HasChunk(ctx, ref)
			if err != nil {
				// give time for the chunk to reach its destination
				time.Sleep(100 * time.Millisecond)
				count++
				continue
			}

			if present {
				break
			}
		}

		// download the chunk from nodeC
		data1, err := nodeC.DownloadChunk(ctx, ref, "")
		if err != nil {
			return err
		}
		if !bytes.Equal(data1, chunk.Data()) {
			return errors.New("chunk downloaded in NodeC does not have proper data")
		}

		// delete the chunk from all nodes. If the chunk from nodeA is not deleted,
		// it is hard to simulate the chunk failure in small clusters. We would need a
		// fairly large cluster then.
		err = deleteChunkFromAllNodes(ctx, ng, chunk)
		if err != nil {
			return err
		}

		// trigger downloading of the chunk from nodeC again (this time it should trigger chunk repair)
		_, err = nodeC.DownloadChunk(ctx, chunk.Address(), addressA.String()[0:2])
		errMessage := fmt.Sprintf("download chunk %s: try again later", chunk.Address().String())
		if err != nil && err.Error() != errMessage { // return error, if chunk recovery is not started
			return fmt.Errorf("chunk recovery not triggered: %w", err)
		}

		// by the time the NodeC creates a trojan chunk and asks NodeA to repair, upload the
		// original chunk in nodeA and pin it
		err = uploadAndPinChunkToNode(ctx, nodeA, chunk)
		if err != nil {
			return err
		}

		count = 0
		t0 := time.Now()
		for {
			if count > maxIterations {
				return fmt.Errorf("could not download even after several attempts")
			}

			// download again to see if the chunk is repaired
			data3, err := nodeC.DownloadChunk(ctx, chunk.Address(), "")
			if err != nil {
				count++
				time.Sleep(1 * time.Second) // give sometime so that the repair happens
				continue                    // if the download is not successful, try again
			}
			d0 := time.Since(t0)

			if !bytes.Equal(data3, chunk.Data()) {
				return errors.New("chunk downloaded in NodeC does not have proper data")
			}

			fmt.Println("repaired chunk ", chunk.Address().String())
			repairedCounter.WithLabelValues(addressA.String()).Inc()
			repairedTimeGauge.WithLabelValues(addressA.String(), chunk.Address().String()).Set(d0.Seconds())
			repairedTimeHistogram.Observe(d0.Seconds())
			break
		}

		if o.MetricsPusher != nil {
			if err := o.MetricsPusher.Push(); err != nil {
				fmt.Printf("chunk %d: %s\n", i, err)
			}
		}
	}

	return nil
}

const (
	maxIterations    = 10
	minNodesRequired = 3
)

var (
	errLessNodesForTest = errors.New("node count is less than the minimum count required")
)

// getNodes get three nodes A, B, C and a chunk such that
// NodeA's and NodeC's first byte of the address does not match
// nodeB is the closest to the generated chunk in the cluster.
func getNodes(ctx context.Context, ng *bee.NodeGroup, rnd *rand.Rand) (*bee.Client, *bee.Client, *bee.Client, *bee.Chunk, error) {
	var overlayA swarm.Address
	var overlayB swarm.Address
	var overlayC swarm.Address
	var chunk *bee.Chunk

	// get overlay addresses of the cluster
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if ng.Size() < minNodesRequired {
		return nil, nil, nil, nil, errLessNodesForTest
	}

	// find node A and C, such that they have the greatest distance between them in the cluster
	overlayA, overlayC, err = findFarthestNodes(overlays)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// find node B
	// generate a chunk and pick the closest address from all the available addresses
	for {
		closestOverlay, c, err := getRandomChunkAndClosestNode(overlays, rnd)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if bytes.Equal(closestOverlay.Bytes(), overlayA.Bytes()) {
			continue
		}
		if bytes.Equal(closestOverlay.Bytes(), overlayC.Bytes()) {
			continue
		}
		// we found our chunk and closest node
		overlayB = closestOverlay
		chunk = c
		break
	}
	fmt.Printf("overlayA: %s\n", overlayA.String())
	fmt.Printf("overlayB: %s\n", overlayB.String())
	fmt.Printf("overlayC: %s\n", overlayC.String())
	fmt.Printf("chunk Address: %s\n", chunk.Address().String())

	// get the nodes for all the addresses
	var nodeA *bee.Client
	var nodeB *bee.Client
	var nodeC *bee.Client
	nodesClients, err := ng.NodesClients(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("get nodes clients: %w", err)
	}
	for _, node := range nodesClients {
		addresses, err := node.Addresses(ctx)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if addresses.Overlay.Equal(overlayA) {
			nodeA = node
		}
		if addresses.Overlay.Equal(overlayB) {
			nodeB = node
		}
		if addresses.Overlay.Equal(overlayC) {
			nodeC = node
		}
	}
	return nodeA, nodeB, nodeC, chunk, nil
}

// uploadAndPinChunkToNode uploads a given chunk to a given node and pins it.
func uploadAndPinChunkToNode(ctx context.Context, node *bee.Client, chunk *bee.Chunk) error {
	ref, err := node.UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
	if err != nil {
		return err
	}

	return node.PinRootHash(ctx, ref)
}

// deleteChunkFromAllNodes deletes a given chunk from al the nodes of the cluster.
func deleteChunkFromAllNodes(ctx context.Context, ng *bee.NodeGroup, chunk *bee.Chunk) error {
	nodesClients, err := ng.NodesClients(ctx)
	if err != nil {
		return fmt.Errorf("get nodes clients: %w", err)
	}

	for _, node := range nodesClients {
		err := node.RemoveChunk(ctx, chunk.Address())
		if err != nil {
			return err
		}
	}
	return nil
}

// getRandomChunkAndClosestNode generates a random node and picks the closest node in the cluster, so that
// when the chunk is uploaded anywhere in the cluster it lands in this node.
func getRandomChunkAndClosestNode(overlays bee.NodeGroupOverlays, rnd *rand.Rand) (swarm.Address, *bee.Chunk, error) {
	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	err = chunk.SetAddress()
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	_, closestAddress, err := chunk.ClosestNodeFromMap(overlays)
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	return closestAddress, &chunk, nil
}

// findFarthestNodes finds two farthest nodes in the cluster
func findFarthestNodes(overlays bee.NodeGroupOverlays) (swarm.Address, swarm.Address, error) {
	var overlayA swarm.Address
	var overlayC swarm.Address
	dist := big.NewInt(0)
	for _, a := range overlays {
		for _, c := range overlays {
			if a.Equal(c) {
				continue
			}
			currDist, err := distance(a.Bytes(), c.Bytes())
			if err != nil {
				return swarm.ZeroAddress, swarm.ZeroAddress, err
			}
			if currDist.Cmp(dist) == 1 {
				dist = currDist
				overlayA = a
				overlayC = c
			}
		}
	}
	return overlayA, overlayC, nil
}

// Distance returns the distance between address x and address y as a (comparable) big integer using the distance metric defined in the swarm specification.
// Fails if not all addresses are of equal length.
func distance(x, y []byte) (*big.Int, error) {
	distanceBytes, err := distanceRaw(x, y)
	if err != nil {
		return nil, err
	}
	r := big.NewInt(0)
	r.SetBytes(distanceBytes)
	return r, nil
}

// DistanceRaw returns the distance between address x and address y in big-endian binary format using the distance metric defined in the swarm specfication.
// Fails if not all addresses are of equal length.
func distanceRaw(x, y []byte) ([]byte, error) {
	if len(x) != len(y) {
		return nil, errors.New("address length must match")
	}
	c := make([]byte, len(x))
	for i, addr := range x {
		c[i] = addr ^ y[i]
	}
	return c, nil
}
