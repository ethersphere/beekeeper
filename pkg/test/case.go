package bee

import (
	"context"
	"math/rand"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type CheckCase struct {
	ctx      context.Context
	clients  map[string]*bee.Client
	cluster  orchestration.Cluster
	overlays orchestration.ClusterOverlays
	logger   logging.Logger

	nodes []BeeV2

	options CaseOptions
	rnd     *rand.Rand
}

type CaseOptions struct {
	FileName      string
	FileSize      int64
	GasPrice      string
	PostageAmount int64
	PostageLabel  string
	Seed          int64
	PostageDepth  uint64
}

func NewCheckCase(ctx context.Context, cluster orchestration.Cluster, caseOpts CaseOptions, logger logging.Logger) (*CheckCase, error) {
	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return nil, err
	}

	overlays, err := cluster.Overlays(ctx)
	if err != nil {
		return nil, err
	}

	logger.Infof("Seed: %d", caseOpts.Seed)

	rnd := random.PseudoGenerator(caseOpts.Seed)

	flatOverlays, err := cluster.FlattenOverlays(ctx)
	if err != nil {
		return nil, err
	}

	rnds := random.PseudoGenerators(caseOpts.Seed, len(flatOverlays))

	var (
		nodes []BeeV2
		count int
	)

	for name, addr := range flatOverlays {
		nodes = append(nodes, BeeV2{
			name:   name,
			Addr:   addr,
			client: clients[name],
			rnd:    rnds[count],
			opts:   caseOpts,
			logger: logger,
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
		options:  caseOpts,
		logger:   logger,
	}, nil
}

func (c *CheckCase) RandomBee() *BeeV2 {
	_, nodeName, overlay := c.overlays.Random(c.rnd)

	return &BeeV2{
		opts:    c.options,
		name:    nodeName,
		overlay: overlay,
		client:  c.clients[nodeName],
		logger:  c.logger,
	}
}

type File struct {
	address swarm.Address
	name    string
	hash    []byte
	rand    *rand.Rand
	size    int64
}

func (c *CheckCase) LastBee() *BeeV2 {
	return &c.nodes[len(c.nodes)-1]
}

func (c *CheckCase) Bee(index int) *BeeV2 {
	return &c.nodes[index]
}

func (c *CheckCase) Balances(ctx context.Context) (balances orchestration.NodeGroupBalances, err error) {
	return c.cluster.FlattenBalances(ctx)
}
