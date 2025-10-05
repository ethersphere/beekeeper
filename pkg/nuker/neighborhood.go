package nuker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/ethersphere/beekeeper/pkg/random"
)

type neighborhoodProvider struct {
	log       logging.Logger
	nodes     node.NodeList
	random    *random.Generator
	useRandom bool
}

func newNeighborhoodProvider(log logging.Logger, nodes node.NodeList, useRandom bool) *neighborhoodProvider {
	if log == nil {
		log = logging.New(io.Discard, 0)
	}

	var randomGen *random.Generator
	if useRandom {
		randomGen = random.NewGenerator(true)
	}

	return &neighborhoodProvider{
		log:       log,
		nodes:     nodes,
		random:    randomGen,
		useRandom: useRandom,
	}
}

func (p *neighborhoodProvider) GetArgs(ctx context.Context, nodeName string, restartArgs []string) ([]string, error) {
	if !p.useRandom {
		return restartArgs, nil
	}

	n := p.nodes.Get(nodeName)
	if n == nil {
		return nil, fmt.Errorf("node %s not found", nodeName)
	}

	response, err := n.Client().Status.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}

	if response.StorageRadius == 0 {
		p.log.Warningf("node %s has storage radius 0, neighborhood will not be set", n.Name())
		return restartArgs, nil
	}

	val, err := p.random.GetRandom(0, (1<<response.StorageRadius)-1)
	if err != nil {
		if errors.Is(err, random.ErrUniqueNumberExhausted) {
			p.log.Warningf("node %s has exhausted all unique neighborhood values", n.Name())
			return restartArgs, nil
		}
		return nil, fmt.Errorf("failed to get random neighborhood: %w", err)
	}

	p.log.Infof("node %s has committed depth %d, using neighborhood value %b", n.Name(), response.StorageRadius, val)

	args := append(restartArgs, fmt.Sprintf("--target-neighborhood=%b", val))

	return args, nil
}

func (p *neighborhoodProvider) UsesRandomNeighborhood() bool {
	return p.useRandom
}
