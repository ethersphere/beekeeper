package orchestration

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
)

type NodeGroup interface {
	Accounting(ctx context.Context) (infos NodeGroupAccounting, err error)
	AddNode(ctx context.Context, name string, inCluster bool, o NodeOptions, opts ...BeeClientOption) (err error)
	Addresses(ctx context.Context) (addrs NodeGroupAddresses, err error)
	Balances(ctx context.Context) (balances NodeGroupBalances, err error)
	DeleteNode(ctx context.Context, name string) (err error)
	DeployNode(ctx context.Context, name string, inCluster bool, o NodeOptions) (ethAddress string, err error)
	GroupReplicationFactor(ctx context.Context, a swarm.Address) (grf int, err error)
	NodeClient(name string) (*bee.Client, error)
	Nodes() map[string]Node
	NodesClients(ctx context.Context) (map[string]*bee.Client, error)
	NodesSorted() (l []string)
	Overlays(ctx context.Context) (overlays NodeGroupOverlays, err error)
	Peers(ctx context.Context) (peers NodeGroupPeers, err error)
	RunningNodes(ctx context.Context) (running []string, err error)
	Settlements(ctx context.Context) (settlements NodeGroupSettlements, err error)
	Size() int
	StoppedNodes(ctx context.Context) (stopped []string, err error)
	Topologies(ctx context.Context) (topologies NodeGroupTopologies, err error)
}

// NodeGroupOptions represents node group options
type NodeGroupOptions struct {
	Annotations               map[string]string
	BeeConfig                 *Config
	Image                     string
	ImagePullPolicy           string
	ImagePullSecrets          []string
	IngressAnnotations        map[string]string
	IngressClass              string
	Labels                    map[string]string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistenceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	ResourcesLimitCPU         string
	ResourcesLimitMemory      string
	ResourcesRequestCPU       string
	ResourcesRequestMemory    string
	UpdateStrategy            string
}

// NodeGroupAddresses represents addresses of all nodes in the node group
type NodeGroupAddresses map[string]bee.Addresses

// NodeGroupAccounting represents accounting of all nodes in the node group
type NodeGroupAccounting map[string]map[string]bee.Account

// NodeGroupBalances represents balances of all nodes in the node group
type NodeGroupBalances map[string]map[string]int64

type FundingOptions struct {
	Eth  float64
	Bzz  float64
	GBzz float64
}

// NodeGroupOverlays represents overlay addresses of all nodes in the node group
type NodeGroupOverlays map[string]swarm.Address

// NodeGroupPeers represents peers of all nodes in the node group
type NodeGroupPeers map[string][]swarm.Address

// NodeGroupSettlements represents settlements of all nodes in the node group
type NodeGroupSettlements map[string]map[string]SentReceived

// SentReceived object
type SentReceived struct {
	Received int64
	Sent     int64
}

// NodeGroupTopologies represents Kademlia topology of all nodes in the node group
type NodeGroupTopologies map[string]bee.Topology

// BeeClientOption represents bee client option
type BeeClientOption func(*bee.ClientOptions) error

// WithURL returns BeeClientOption with given api url
func WithURL(apiURL string) BeeClientOption {
	return func(o *bee.ClientOptions) error {
		api, err := url.Parse(apiURL)
		if err != nil {
			return fmt.Errorf("invalid api url: %w", err)
		}

		o.APIURL = api
		return nil
	}
}

// WithNoOptions represents no BeeClientOption
func WithNoOptions() BeeClientOption {
	return func(o *bee.ClientOptions) error {
		return nil
	}
}
