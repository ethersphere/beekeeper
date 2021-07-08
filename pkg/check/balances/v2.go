package balances

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/random"
)

func (c *Check) RunV2(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.DryRun {
		fmt.Println("running balances (dry mode)")
		return dryRun(ctx, cluster, o)
	}

	fmt.Println("running balances")

	var clusterV2 *clusterV2

	if clusterV2, err = newClusterV2(ctx, cluster, o); err != nil {
		return err
	}

	if err := clusterV2.SaveBalances(); err != nil {
		return err
	}

	if err := validateInitialBalances(ctx, clusterV2, o); err != nil {
		return err
	}

	// repeats
	for i := 0; i < o.UploadNodeCount; i++ {
		// upload/check
		node := clusterV2.RandomNode()

		file := bee.NewRandomFile(clusterV2.rnd, fmt.Sprintf("%s-%s", o.FileName, node.name), o.FileSize)

		if err := node.UploadFile(ctx, &file, o); err != nil {
			return err
		}
		if err := clusterV2.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}

		// download/check
		if err := clusterV2.RandomNode().ExpectToHaveFile(ctx, file); err != nil {
			return err
		}
		if err := clusterV2.SaveBalances(); err != nil {
			return err
		}
		if err := clusterV2.ExpectBalancesHaveChanged(ctx); err != nil {
			return err
		}
	}

	return nil
}

func validateInitialBalances(ctx context.Context, cluster *clusterV2, opts Options) error {
	flatBalances := cluster.prevBalances()
	flatOverlays := flattenOverlays(cluster.overlays)

	if err := validateBalances(flatOverlays, flatBalances); err != nil {
		return fmt.Errorf("invalid initial balances: %s", err.Error())
	}

	fmt.Println("Balances are valid")

	return nil
}

type clusterV2 struct {
	ctx context.Context
	// cluster         *bee.Cluster
	balances        func(ctx context.Context) (balances bee.ClusterBalances, err error)
	clients         map[string]*bee.Client
	overlays        bee.ClusterOverlays
	balancesHistory []bee.NodeGroupBalances
	rnd             *rand.Rand
}

func newClusterV2(ctx context.Context, cluster *bee.Cluster, o Options) (*clusterV2, error) {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, err
	}

	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return nil, err
	}

	rnd := random.PseudoGenerator(o.Seed)
	fmt.Printf("Seed: %d\n", o.Seed)

	return &clusterV2{
		ctx:      ctx,
		clients:  clients,
		overlays: overlays,
		rnd:      rnd,
		balances: func(ctx context.Context) (bee.ClusterBalances, error) {
			return cluster.Balances(ctx)
		},
	}, nil
}

func (c *clusterV2) SaveBalances() error {
	balances, err := c.balances(c.ctx)

	if err != nil {
		return err
	}

	flatBalances := flattenBalances(balances)

	c.balancesHistory = append(c.balancesHistory, flatBalances)

	return nil
}

type nodev2 struct {
	name    string
	overlay swarm.Address
	client  *bee.Client
}

func (c *clusterV2) RandomNode() *nodev2 {
	_, nodeName, overlay := c.overlays.Random(c.rnd)

	return &nodev2{
		name:    nodeName,
		overlay: overlay,
		client:  c.clients[nodeName],
	}
}

func (n nodev2) UploadFile(ctx context.Context, file *bee.File, o Options) error {
	depth := 2 + bee.EstimatePostageBatchDepth(file.Size())
	batchID, err := n.client.CreatePostageBatch(ctx, o.PostageAmount, depth, o.GasPrice, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: created batch id %w", n.name, err)
	}
	fmt.Printf("node %s: created batch id %s\n", n.name, batchID)
	time.Sleep(o.PostageWait)

	if err := n.client.UploadFile(ctx, file, api.UploadOptions{BatchID: batchID}); err != nil {
		return fmt.Errorf("node %s: %w", n.name, err)
	}

	fmt.Println("Uploaded file to", n.name)

	return nil
}

func (n nodev2) ExpectToHaveFile(ctx context.Context, file bee.File) error {
	size, hash, err := n.client.DownloadFile(ctx, file.Address())
	if err != nil {
		return fmt.Errorf("node %s: %w", n.name, err)
	}

	fmt.Println("Downloaded file from", n.name)

	if !bytes.Equal(file.Hash(), hash) {
		return fmt.Errorf("file %s not retrieved successfully from node %s. Uploaded size: %d Downloaded size: %d", file.Address().String(), n.name, file.Size(), size)
	}

	return nil
}

func (c *clusterV2) prevBalances() bee.NodeGroupBalances {
	len := len(c.balancesHistory)
	return c.balancesHistory[len-1]
}

func (c *clusterV2) ExpectBalancesHaveChanged(ctx context.Context) error {
	for t := 0; t < 5; t++ {
		time.Sleep(2 * time.Duration(t) * time.Second)

		balances, err := c.balances(c.ctx)

		if err != nil {
			return err
		}

		flatBalances := flattenBalances(balances)
		balancesHaveChanged(flatBalances, c.prevBalances())

		flatOverlays := flattenOverlays(c.overlays)

		if err := expectCreditsEqualDebits(flatOverlays, flatBalances); err != nil {
			fmt.Println("Invalid balances after downloading a file:", err)
			fmt.Println("Retrying ...", t)
			continue
		}

		fmt.Println("Balances are valid")

		break
	}

	return nil
}

func expectCreditsEqualDebits(overlays map[string]swarm.Address, balances map[string]map[string]int64) (err error) {
	return validateBalances(overlays, balances)
}
