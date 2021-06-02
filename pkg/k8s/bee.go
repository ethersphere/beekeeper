package k8s

import (
	"context"
	"errors"
)

// ErrNotSet represents error when Kubernetes Bee client is not set
var ErrNotSet = errors.New("kubernetes Bee client not set")

// Bee represents Bee implementation in Kubernetes
type Bee interface {
	Create(ctx context.Context, o CreateOptions) (err error)
	Delete(ctx context.Context, name, namespace string) (err error)
	Ready(ctx context.Context, name, namespace string) (ready bool, err error)
	RunningNodes(ctx context.Context, namespace string) (running []string, err error)
	Start(ctx context.Context, name, namespace string) (err error)
	Stop(ctx context.Context, name, namespace string) (err error)
	StoppedNodes(ctx context.Context, namespace string) (stopped []string, err error)
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
	APIAddr                    string // HTTP API listen address
	BlockTime                  uint64 // chain block time
	Bootnodes                  string // initial nodes to connect to
	BootnodeMode               bool   // cause the node to always accept incoming connections
	CacheCapacity              uint64 // cache capacity in chunks, multiply by 4096 (MaxChunkSize) to get approximate capacity in bytes
	ClefSignerEnable           bool   // enable clef signer
	ClefSignerEndpoint         string // clef signer endpoint
	CORSAllowedOrigins         string // origins with CORS headers enabled
	DataDir                    string // data directory
	DbOpenFilesLimit           int    // number of open files allowed by database
	DbBlockCacheCapacity       int    // size of block cache of the database in bytes
	DbWriteBufferSize          int    // size of the database write buffer in bytes
	DbDisableSeeksCompaction   bool   // disables DB compactions triggered by seeks
	DebugAPIAddr               string // debug HTTP API listen address
	DebugAPIEnable             bool   // enable debug HTTP API
	FullNode                   bool   // cause the node to start in full mode
	GatewayMode                bool   // disable a set of sensitive features in the api
	GlobalPinningEnabled       bool   // enable global pinning
	NATAddr                    string // NAT exposed address
	NetworkID                  uint64 // ID of the Swarm network
	P2PAddr                    string // P2P listen address
	P2PQUICEnable              bool   // enable P2P QUIC transport
	P2PWSEnable                bool   // enable P2P WebSocket transport
	Password                   string // password for decrypting keys
	PaymentEarly               uint64 // amount in BZZ below the peers payment threshold when we initiate settlement
	PaymentThreshold           uint64 // threshold in BZZ where you expect to get paid from your peers
	PaymentTolerance           uint64 // excess debt above payment threshold in BZZ where you disconnect from your peer
	PostageStampAddress        string // postage stamp address
	PriceOracleAddress         string // price Oracle address
	ResolverOptions            string // ENS compatible API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url
	Standalone                 bool   // whether we want the node to start with no listen addresses for p2p
	SwapEnable                 bool   // enable swap
	SwapEndpoint               string // swap ethereum blockchain endpoint
	SwapFactoryAddress         string // swap factory address
	SwapLegacyFactoryAddresses string // swap legacy factory addresses
	SwapInitialDeposit         uint64 // initial deposit if deploying a new chequebook
	TracingEnabled             bool   // enable tracing
	TracingEndpoint            string // endpoint to send tracing data
	TracingServiceName         string // service name identifier for tracing
	Verbosity                  uint64 // log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace
	WelcomeMessage             string // send a welcome message string during handshakes
}
