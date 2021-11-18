package bee

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type CheckCase struct {
	ctx      context.Context
	clients  map[string]*bee.Client
	cluster  orchestration.Cluster
	overlays orchestration.ClusterOverlays

	nodes []beeV2

	options CaseOptions
	rnd     *rand.Rand
}

type CaseOptions struct {
	FileName      string
	FileSize      int64
	GasPrice      string
	PostageAmount int64
	PostageLabel  string
	PostageWait   time.Duration
	Seed          int64
	PostageDepth  uint64
}

func NewCheckCase(ctx context.Context, cluster orchestration.Cluster, o CaseOptions) (*CheckCase, error) {
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

	flatOverlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return nil, err
	}

	rnds := random.PseudoGenerators(o.Seed, len(flatOverlays))
	fmt.Printf("Seed: %d\n", o.Seed)

	var (
		nodes []beeV2
		count int
	)
	for name, addr := range flatOverlays {
		nodes = append(nodes, beeV2{
			name:   name,
			Addr:   addr,
			client: clients[name],
			rnd:    rnds[count],
			o:      o,
		})
		count++
	}

	return &CheckCase{
		ctx:      ctx,
		clients:  clients,
		cluster:  cluster,
		overlays: overlays,
		nodes:    nodes,
		rnd:      rnd,
		options:  o,
	}, nil
}

func (c *CheckCase) RandomBee() *beeV2 {
	_, nodeName, overlay := c.overlays.Random(c.rnd)

	return &beeV2{
		o:       c.options,
		name:    nodeName,
		overlay: overlay,
		client:  c.clients[nodeName],
	}
}

type File struct {
	address swarm.Address
	name    string
	hash    []byte
	rand    *rand.Rand
	size    int64
}

func (c *CheckCase) LastBee() *beeV2 {
	return &c.nodes[len(c.nodes)-1]
}

func (c *CheckCase) Bee(index int) *beeV2 {
	return &c.nodes[index]
}

func (c *CheckCase) Balances(ctx context.Context) (balances orchestration.NodeGroupBalances, err error) {
	return c.cluster.FlattenBalances(ctx)
}
