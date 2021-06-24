package betakappa

type K8S struct{}

func (*K8S) NewCluster() *Cluster { return &Cluster{} }

type (
	Cluster         []NodeGroup
	NodeGroupConfig struct{}
)

func (*Cluster) Add(*NodeGroupConfig) *NodeGroup     { return nil }
func (*Cluster) ShutDown() error                     { return nil }
func (*Cluster) ForEachNode(func(*Node) error) error { return nil } // convenient method
func (*Cluster) peers() []string                     { return nil } // internal

type (
	NodeGroup  []Node
	NodeConfig struct{}
)

func (*NodeGroup) Add(*NodeConfig) *Node      { return nil }
func (*NodeGroup) Parent() Cluster            { return nil }
func (*NodeGroup) RunningNodes() RunningNodes { return nil }

type Node struct{}

func (*Node) Start() error              { return nil }
func (*Node) RunningNode() *RunningNode { return nil }
func (*Node) Remove() error             { return nil }
func (*Node) podName() *string          { return nil } // internal
func (*Node) podConfig() *string        { return nil }

type RunningNode struct{}

func (*RunningNode) Stop() error        { return nil }
func (*RunningNode) Fund() error        { return nil }
func (*RunningNode) Bee() *Bee          { return nil }
func (*RunningNode) Parent() *NodeGroup { return nil }

type RunningNodes []RunningNode

func (RunningNodes) Bees() Bees                            { return nil }
func (RunningNodes) Filter(critiria struct{}) RunningNodes { return nil }
func (RunningNodes) peers() []string                       { return nil } // internal
func (RunningNodes) overlay() *string                      { return nil }

type Bee struct{} // has the http client

func (*Bee) Upload(chunk struct{}) error { return nil }
func (*Bee) PssMessageTo(*Bee) error     { return nil }
func (*Bee) GetOrCreateBatch(*Bee) error { return nil }
func (*Bee) address()                    {}
func (*Bee) balances()                   {}
func (*Bee) overlay() *string            { return nil }
func (*Bee) peers()                      {}
func (*Bee) settlements()                {}

type Bees []Bee

func (Bees) PickRandom(excluding ...*Bee) *Bee { return nil }

// utility functions
func RandomChunk() struct{} { return struct{}{} }

type PeerSet interface {
	peers() []string
}

func ClosestPeer(set PeerSet, b *Bee) *RunningNode {
	_ = set.peers()
	/* lookup closest */
	return nil
}

// allows for consistent reuse
func ExpectReceivedPssMsg(from, to *RunningNode) bool                  { return false }
func ExpectPeerInBin(node *RunningNode, in *RunningNode, bin int) bool { return false }
func ExpectToHaveFile(file struct{}, nodes ...RunningNode) bool        { return false }
func ExpectHasChunk(chunk struct{}, nodes ...RunningNode) bool         { return false }
