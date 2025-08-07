package nuker

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/ethersphere/beekeeper/pkg/random"
	v1 "k8s.io/api/apps/v1"
)

// NeighborhoodArgProvider defines how to get extra restart args for a StatefulSet.
type NeighborhoodArgProvider interface {
	GetArgs(ctx context.Context, ss *v1.StatefulSet, restartArgs []string) ([]string, error)
}

type randomNeighborhoodProvider struct {
	log    logging.Logger
	nodes  node.NodeList
	random *random.Generator
}

func NewRandomNeighborhoodProvider(log logging.Logger, nodes node.NodeList) NeighborhoodArgProvider {
	if log == nil {
		log = logging.New(io.Discard, 0)
	}

	return &randomNeighborhoodProvider{
		log:    log,
		nodes:  nodes,
		random: random.NewGenerator(true),
	}
}

func (p *randomNeighborhoodProvider) GetArgs(ctx context.Context, ss *v1.StatefulSet, restartArgs []string) ([]string, error) {
	podNames := getPodNames(ss)
	if len(podNames) != 1 {
		return nil, errors.New("random neighborhood provider requires exactly one pod in the StatefulSet")
	}

	node := p.nodes.Get(podNames[0])
	if node == nil {
		return nil, errors.New("failed to get node for pod")
	}

	response, err := node.Client().Status.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}

	val, err := p.random.GetRandom(0, 1<<response.StorageRadius) // TODO: check if this is correct property to use for neighborhood value
	if err != nil {
		return nil, fmt.Errorf("failed to get random neighborhood: %w", err)
	}

	p.log.Infof("node %s has committed depth %d, using neighborhood value %b", node.Name(), response.StorageRadius, val)

	args := append(restartArgs, fmt.Sprintf("--target-neighborhood=%b", val))

	return args, nil
}
