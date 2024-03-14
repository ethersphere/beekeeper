package nokube

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
)

// compile check whether client implements interface
var _ orchestration.Node = (*Node)(nil)

// Node represents Bee node
type Node struct {
	*k8s.Node
}

// NewNode returns Bee node
func NewNode(name string, opts orchestration.NodeOptions, logger logging.Logger) (n *Node) {
	return &Node{
		Node: k8s.NewNode(name, opts, logger),
	}
}

func (n Node) Create(ctx context.Context, o orchestration.CreateOptions) (err error) {
	panic("unimplemented")
}

func (n Node) Delete(ctx context.Context, namespace string) (err error) {
	panic("unimplemented")
}

func (n Node) Ready(ctx context.Context, namespace string) (ready bool, err error) {
	panic("unimplemented")
}

func (n Node) Start(ctx context.Context, namespace string) (err error) {
	panic("unimplemented")
}

func (n Node) Stop(ctx context.Context, namespace string) (err error) {
	panic("unimplemented")
}
