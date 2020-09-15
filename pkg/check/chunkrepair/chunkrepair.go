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
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check/fullconnectivity"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents chunk repair check options
type Options struct {
	NumberOfChunksToRepair int
	Seed                   int64
}

const (
	maxIterations    = 10
	minNodesRequired = 3
)

var (
	errLessNodesForTest = errors.New("node count is less than the minimum count required")
	errFullConnectivity = errors.New("full connectivity, chunk repair cannot be tested")
)

// Check ...
func Check(c bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {
	ctx := context.Background()
	rnds := random.PseudoGenerators(o.Seed, o.NumberOfChunksToRepair)
	fmt.Printf("Seed: %d\n", o.Seed)

	pusher.Collector(repairedCounter)
	pusher.Collector(repairedTimeGauge)
	pusher.Collector(repairedTimeHistogram)

	pusher.Format(expfmt.FmtText)

	if err := fullconnectivity.Check(c); err == nil {
		return errFullConnectivity
	}

	// for every chunk
	// 1) Upload the chunk in NodeA
	// 2) check if it reached nodeB
	// 3) download the chunk from NodeC
	// 4) delete chunk from all nodes except nodeA
	// 5) Now try downloading with nodeA as target (we should get the repaired chunk)
	// 6) download the chunk again without target (it should succeed now as chunk repairing would have placed the chunk in proper node in the network)
	for i := 0; i < o.NumberOfChunksToRepair; i++ {
		// Pick node A, B, C and a chunk which is closest to B
		nodeA, nodeB, nodeC, chunk, err := getNodes(ctx, c, rnds[i])
		if err != nil {
			return err
		}
		addressA, err := nodeA.Overlay(ctx)
		if err != nil {
			return err
		}

		// 1) upload the chunk in nodeA
		err = nodeA.UploadChunk(ctx, chunk)
		if err != nil {
			return err
		}

		// 2) checking if the node reached nodeB
		count := 0
		for {
			if count > maxIterations {
				return fmt.Errorf("could not get chunk even after several attempts")
			}

			// check if the node is there in the local store of node B
			// this does a get chunk instead of Has chunk, so the following
			// call just checks if the chunk is accessible from nodeB
			present, err := nodeB.HasChunk(ctx, chunk.Address())
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

		// 3) download the chunk from nodeC ( it should work as NodeB has the chunk)
		data1, err := nodeC.DownloadChunk(ctx, chunk.Address(), "")
		if err != nil {
			return err
		}
		if !bytes.Equal(data1, chunk.Data()) {
			return errors.New("chunk downloaded in NodeC does not have proper data")
		}

		// 4) delete the chunk from all nodes except nodeA
		var nodes []bee.Node
		exceptNodes := append(nodes, nodeA)
		err = deleteChunkFromAllNodesExceptNode(ctx, c, chunk, exceptNodes)
		if err != nil {
			return err
		}

		// 5) Now trigger downloading of the chunk from nodeC again (this time it should trigger chunk repair )
		var d0 time.Duration
		t0 := time.Now()
		data3, err := nodeC.DownloadChunk(ctx, chunk.Address(), addressA.String()[0:2])
		d0 = time.Since(t0)
		if err != nil {
			return fmt.Errorf("chunk recovery not triggered: %w", err)
		}

		if !bytes.Equal(data3, chunk.Data()) {
			return fmt.Errorf("chunk recovery failed")
		}

		fmt.Println("got chunk in ", chunk.Address().String(), d0.String())
		if d0.Seconds() < (50 * time.Second).Seconds() {
			return fmt.Errorf("chunk was not obtained through repairing")
		}

		repairedCounter.WithLabelValues(addressA.String()).Inc()
		repairedTimeGauge.WithLabelValues(addressA.String(), chunk.Address().String()).Set(d0.Seconds())
		repairedTimeHistogram.Observe(d0.Seconds())

		count = 0

		for {
			if count > maxIterations {
				return fmt.Errorf("could not download even after several attempts")
			}

			// 6) download again from nodeC without targets to see if the chunk is repaired in the network
			data3, err := nodeC.DownloadChunk(ctx, chunk.Address(), "")
			if err != nil {
				count++
				time.Sleep(1 * time.Second) // give sometime so that the repair happens
				continue                    // if the download is not successful, try again
			}

			if !bytes.Equal(data3, chunk.Data()) {
				return errors.New("chunk downloaded in NodeC does not have proper data")
			}
			fmt.Println("chunk repaired in network", chunk.Address().String())
			break
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
// nodeB is the closest to the generated chunk in the cluster.
func getNodes(ctx context.Context, c bee.Cluster, rnd *rand.Rand) (bee.Node, bee.Node, bee.Node, *bee.Chunk, error) {
	var overlayA swarm.Address
	var overlayB swarm.Address
	var overlayC swarm.Address
	var chunk *bee.Chunk

	// get overlay addresses of the cluster
	overlays, err := c.Overlays(ctx)
	if err != nil {
		return bee.Node{}, bee.Node{}, bee.Node{}, nil, err
	}

	if len(overlays) < minNodesRequired {
		return bee.Node{}, bee.Node{}, bee.Node{}, nil, errLessNodesForTest
	}

	// find node B
	// generate a chunk and pick the closest address from all the available addresses
	overlayB, chunk, err = getRandomChunkAndClosestNode(overlays, rnd)
	if err != nil {
		return bee.Node{}, bee.Node{}, bee.Node{}, nil, err
	}

	peers, err := c.Peers(ctx)
	if err != nil {
		return bee.Node{}, bee.Node{}, bee.Node{}, nil, err
	}

	// find node A and C, such that they have the greatest distance between them in the cluster
	overlayA, overlayC, err = findFarthestNodes(overlays, peers, overlayB)
	if err != nil {
		return bee.Node{}, bee.Node{}, bee.Node{}, nil, err
	}

	fmt.Printf("overlayA: %s\n", overlayA.String())
	fmt.Printf("overlayB: %s\n", overlayB.String())
	fmt.Printf("overlayC: %s\n", overlayC.String())
	fmt.Printf("chunk Address: %s\n", chunk.Address().String())

	// get the nodes for all the addresses
	var nodeA bee.Node
	var nodeB bee.Node
	var nodeC bee.Node
	for _, node := range c.Nodes {
		addresses, err := node.Addresses(ctx)
		if err != nil {
			return bee.Node{}, bee.Node{}, bee.Node{}, nil, err
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

// deleteChunkFromAllNodesExceptNode deletes a given chunk from al the nodes of the cluster.
func deleteChunkFromAllNodesExceptNode(ctx context.Context, c bee.Cluster, chunk *bee.Chunk, exceptNodes []bee.Node) error {
	for _, node := range c.Nodes {
		overlay, err := node.Overlay(ctx)
		if err != nil {
			return err
		}
		for _, exceptNode := range exceptNodes {
			exceptOverlay, err := exceptNode.Overlay(ctx)
			if err != nil {
				return err
			}
			if !overlay.Equal(exceptOverlay) {
				err := node.RemoveChunk(ctx, chunk)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getRandomChunkAndClosestNode generates a random node and picks the closest node in the cluster, so that
// when the chunk is uploaded anywhere in the cluster it lands in this node.
func getRandomChunkAndClosestNode(overlays []swarm.Address, rnd *rand.Rand) (swarm.Address, *bee.Chunk, error) {
	chunk, err := bee.NewRandomChunk(rnd)
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	err = chunk.SetAddress()
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	closestAddress, err := chunk.ClosestNode(overlays)
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	return closestAddress, &chunk, nil
}

// findFarthestNodes finds two farthest nodes in the cluster
func findFarthestNodes(overlays []swarm.Address, peers [][]swarm.Address, overlayB swarm.Address) (swarm.Address, swarm.Address, error) {
	var overlayA swarm.Address
	var overlayC swarm.Address
	dist := big.NewInt(0)

	for _, ps := range peers {
		// if the peer is connected to all the peers, then ignore it
		if len(ps) == len(peers)-1 {
			continue
		}

		// if the peer does not contain overlayB in its peer list, then ignore it
		if !contains(ps, overlayB) {
			continue
		}

		// find the farthest peer and also the peer should not be in the neighbour list
		for _, c := range overlays {
			for _, a := range ps {
				if a.Equal(c) {
					continue
				}
				currDist, err := distance(a.Bytes(), c.Bytes())
				if err != nil {
					return swarm.ZeroAddress, swarm.ZeroAddress, err
				}
				if currDist.Cmp(dist) == 1 && !contains(ps, c) {
					dist = currDist
					overlayA = a
					overlayC = c
				}
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

// contains checks if a given set of swarm.Address contains given swarm.Address
func contains(s []swarm.Address, v swarm.Address) bool {
	for _, a := range s {
		if a.Equal(v) {
			return true
		}
	}
	return false
}
