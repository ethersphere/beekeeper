package orchestration

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
)

// ErrNotSet represents error when orchestration client is not set
var ErrNotSet = errors.New("orchestration client not set")

type Node interface {
	Name() string
	Client() *bee.Client
	ClefKey() string
	ClefPassword() string
	Config() *Config
	Create(ctx context.Context, o CreateOptions) (err error)
	Delete(ctx context.Context, namespace string) (err error)
	LibP2PKey() string
	Ready(ctx context.Context, namespace string) (ready bool, err error)
	Start(ctx context.Context, namespace string) (err error)
	Stop(ctx context.Context, namespace string) (err error)
	SwarmKey() string
	SetSwarmKey(key string) Node
	SetClefKey(key string) Node
	SetClefPassword(key string) Node
}

// NodeOptions holds optional parameters for the Node.
type NodeOptions struct {
	ClefKey      string
	ClefPassword string
	Client       *bee.Client
	Config       *Config
	K8S          *k8s.Client
	LibP2PKey    string
	SwarmKey     string
}

// CreateOptions represents available options for creating node
type CreateOptions struct {
	// Bee configuration
	Config Config
	// Kubernetes configuration
	Name                      string
	Namespace                 string
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	ClefKey                   string
	ClefPassword              string
	Labels                    map[string]string
	Image                     string
	ImagePullPolicy           string
	ImagePullSecrets          []string
	IngressAnnotations        map[string]string
	IngressClass              string
	IngressHost               string
	IngressDebugAnnotations   map[string]string
	IngressDebugClass         string
	IngressDebugHost          string
	LibP2PKey                 string
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
	Selector                  map[string]string
	SwarmKey                  string
	UpdateStrategy            string
}

// Config represents Bee configuration
type Config struct {
	AllowPrivateCIDRs          bool          // allow to advertise private CIDRs to the public network
	APIAddr                    string        // HTTP API listen address
	BlockTime                  uint64        // chain block time
	Bootnodes                  string        // initial nodes to connect to
	BootnodeMode               bool          // cause the node to always accept incoming connections
	CacheCapacity              uint64        // cache capacity in chunks, multiply by 4096 (MaxChunkSize) to get approximate capacity in bytes
	ClefSignerEnable           bool          // enable clef signer
	ClefSignerEndpoint         string        // clef signer endpoint
	CORSAllowedOrigins         string        // origins with CORS headers enabled
	DataDir                    string        // data directory
	DbOpenFilesLimit           int           // number of open files allowed by database
	DbBlockCacheCapacity       int           // size of block cache of the database in bytes
	DbWriteBufferSize          int           // size of the database write buffer in bytes
	DbDisableSeeksCompaction   bool          // disables DB compactions triggered by seeks
	FullNode                   bool          // cause the node to start in full mode
	GatewayMode                bool          // disable a set of sensitive features in the api
	GlobalPinningEnabled       bool          // enable global pinning
	Mainnet                    bool          // enable mainnet
	NATAddr                    string        // NAT exposed address
	NetworkID                  uint64        // ID of the Swarm network
	P2PAddr                    string        // P2P listen address
	P2PQUICEnable              bool          // enable P2P QUIC transport
	P2PWSEnable                bool          // enable P2P WebSocket transport
	Password                   string        // password for decrypting keys
	PaymentEarly               uint64        // amount in BZZ below the peers payment threshold when we initiate settlement
	PaymentThreshold           uint64        // threshold in BZZ where you expect to get paid from your peers
	PaymentTolerance           uint64        // excess debt above payment threshold in BZZ where you disconnect from your peer
	PostageStampAddress        string        // postage stamp address
	PriceOracleAddress         string        // price Oracle address
	ResolverOptions            string        // ENS compatible API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url
	Restricted                 bool          // start node in restricted mode
	TokenEncryptionKey         string        // username for API authentication
	AdminPassword              string        // password hash for API authentication
	Standalone                 bool          // whether we want the node to start with no listen addresses for p2p
	ChequebookEnable           bool          // enable chequebook
	SwapEnable                 bool          // enable swap
	SwapEndpoint               string        // swap ethereum blockchain endpoint
	SwapDeploymentGasPrice     string        // gas price in wei to use for deployment and funding
	SwapFactoryAddress         string        // swap factory address
	SwapLegacyFactoryAddresses string        // swap legacy factory addresses
	SwapInitialDeposit         uint64        // initial deposit if deploying a new chequebook
	TracingEnabled             bool          // enable tracing
	TracingEndpoint            string        // endpoint to send tracing data
	TracingServiceName         string        // service name identifier for tracing
	Transaction                string        // proof-of-identity transaction hash
	Verbosity                  uint64        // log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace
	WelcomeMessage             string        // send a welcome message string during handshakes
	WarmupTime                 time.Duration // warmup time pull/pushsync protocols
}
