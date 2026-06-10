package orchestration

import (
	"context"
	"errors"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

var ErrNotSet = errors.New("orchestration client not set")

type Node interface {
	Client() *bee.Client
	Config() *Config
	LibP2PKey() string
	Name() string
	SetSwarmKey(key *EncryptedKey) Node
	SwarmKey() *EncryptedKey
	Create(ctx context.Context, o CreateOptions) (err error)
	Delete(ctx context.Context, namespace string) (err error)
	Ready(ctx context.Context, namespace string) (ready bool, err error)
	Start(ctx context.Context, namespace string) (err error)
	Stop(ctx context.Context, namespace string) (err error)
}

type NodeOrchestrator interface {
	Create(ctx context.Context, o CreateOptions) (err error)
	Delete(ctx context.Context, name string, namespace string) (err error)
	Ready(ctx context.Context, name string, namespace string) (ready bool, err error)
	Start(ctx context.Context, name string, namespace string) (err error)
	Stop(ctx context.Context, name string, namespace string) (err error)
	RunningNodes(ctx context.Context, namespace string) (running []string, err error)
	StoppedNodes(ctx context.Context, namespace string) (stopped []string, err error)
}

// NodeOptions holds optional parameters for the Node.
type NodeOptions struct {
	Config    *Config
	LibP2PKey string
	SwarmKey  *EncryptedKey
}

// CreateOptions represents available options for creating node
type CreateOptions struct {
	// Bee configuration
	Config Config
	// Kubernetes configuration
	Name                      string
	Namespace                 string
	Annotations               map[string]string
	Labels                    map[string]string
	Image                     string
	ImagePullPolicy           string
	ImagePullSecrets          []string
	IngressAnnotations        map[string]string
	IngressClass              string
	IngressHost               string
	LibP2PKey                 string
	NodeSelector              map[string]string
	P2PWSSNodePort            int32
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
	SwarmKey                  *EncryptedKey
	UpdateStrategy            string
}

// Config represents Bee configuration.
//
// Every field is a pointer so that a nil value means "not set in Beekeeper's
// YAML config". Such fields are omitted from the rendered .bee.yaml (see the
// yaml tags below), which lets the Bee node fall back to its own built-in
// default. A non-nil pointer is always rendered, even when it points to a zero
// value (e.g. full-node: false, warmup-time: 0s), so explicitly configured zero
// values still reach Bee. Beekeeper never hardcodes any of Bee's defaults.
//
// The yaml tags are the Bee flag names and are only used when marshalling this
// struct into the node's .bee.yaml. A few flag names differ from Beekeeper's
// config keys (e.g. bootnode, tracing-enable); those input keys live on
// config.BeeConfig and are unaffected.
type Config struct {
	AllowPrivateCIDRs           *bool          `yaml:"allow-private-cidrs,omitempty"`            // allow to advertise private CIDRs to the public network
	APIAddr                     *string        `yaml:"api-addr,omitempty"`                       // HTTP API listen address
	AutoTLSCAEndpoint           *string        `yaml:"autotls-ca-endpoint,omitempty"`            // autotls certificate authority endpoint
	AutoTLSDomain               *string        `yaml:"autotls-domain,omitempty"`                 // autotls domain
	AutoTLSRegistrationEndpoint *string        `yaml:"autotls-registration-endpoint,omitempty"`  // autotls registration endpoint
	BlockchainRPCDialTimeout    *time.Duration `yaml:"blockchain-rpc-dial-timeout,omitempty"`    // blockchain rpc TCP dial timeout
	BlockchainRPCEndpoint       *string        `yaml:"blockchain-rpc-endpoint,omitempty"`        // rpc blockchain endpoint
	BlockchainRPCIdleTimeout    *time.Duration `yaml:"blockchain-rpc-idle-timeout,omitempty"`    // blockchain rpc idle connection timeout
	BlockchainRPCKeepalive      *time.Duration `yaml:"blockchain-rpc-keepalive,omitempty"`       // blockchain rpc TCP keepalive interval
	BlockchainRPCTLSTimeout     *time.Duration `yaml:"blockchain-rpc-tls-timeout,omitempty"`     // blockchain rpc TLS handshake timeout
	BlockSyncInterval           *uint64        `yaml:"block-sync-interval,omitempty"`            // block number cache sync interval in blocks
	BlockTime                   *uint64        `yaml:"block-time,omitempty"`                     // chain block time
	BootnodeMode                *bool          `yaml:"bootnode-mode,omitempty"`                  // cause the node to always accept incoming connections
	Bootnodes                   *[]string      `yaml:"bootnode,omitempty"`                       // initial nodes to connect to
	CacheCapacity               *uint64        `yaml:"cache-capacity,omitempty"`                 // cache capacity in chunks, multiply by 4096 to get approximate capacity in bytes
	CacheRetrieval              *bool          `yaml:"cache-retrieval,omitempty"`                // enable forwarded content caching
	ChequebookEnable            *bool          `yaml:"chequebook-enable,omitempty"`              // enable chequebook
	ChequebookMinBalance        *string        `yaml:"chequebook-min-balance,omitempty"`         // minimum chequebook token balance required for verification, in token small units
	ChequebookVerification      *bool          `yaml:"chequebook-verification,omitempty"`        // reject full-node hive/handshake records that carry no chequebook address
	CORSAllowedOrigins          *[]string      `yaml:"cors-allowed-origins,omitempty"`           // origins with CORS headers enabled
	DataDir                     *string        `yaml:"data-dir,omitempty"`                       // data directory
	DbBlockCacheCapacity        *uint64        `yaml:"db-block-cache-capacity,omitempty"`        // size of block cache of the database in bytes
	DbDisableSeeksCompaction    *bool          `yaml:"db-disable-seeks-compaction,omitempty"`    // disables db compactions triggered by seeks
	DbOpenFilesLimit            *uint64        `yaml:"db-open-files-limit,omitempty"`            // number of open files allowed by database
	DbWriteBufferSize           *uint64        `yaml:"db-write-buffer-size,omitempty"`           // size of the database write buffer in bytes
	FullNode                    *bool          `yaml:"full-node,omitempty"`                      // cause the node to start in full mode
	GasLimitFallback            *uint64        `yaml:"gas-limit-fallback,omitempty"`             // gas limit fallback when estimation fails for contract transactions
	Mainnet                     *bool          `yaml:"mainnet,omitempty"`                        // triggers connect to main net bootnodes
	MinimumGasTipCap            *uint64        `yaml:"minimum-gas-tip-cap,omitempty"`            // minimum gas tip cap in wei for transactions, 0 means use suggested gas tip cap
	MinimumStorageRadius        *uint          `yaml:"minimum-storage-radius,omitempty"`         // minimum radius storage threshold
	NATAddr                     *string        `yaml:"nat-addr,omitempty"`                       // NAT exposed address
	NATWSSAddr                  *string        `yaml:"nat-wss-addr,omitempty"`                   // WSS NAT exposed address
	NeighborhoodSuggester       *string        `yaml:"neighborhood-suggester,omitempty"`         // suggester for target neighborhood
	NetworkID                   *uint64        `yaml:"network-id,omitempty"`                     // ID of the Swarm network
	P2PAddr                     *string        `yaml:"p2p-addr,omitempty"`                       // P2P listen address
	P2PWSEnable                 *bool          `yaml:"p2p-ws-enable,omitempty"`                  // enable P2P WebSocket transport
	P2PWSSAddr                  *string        `yaml:"p2p-wss-addr,omitempty"`                   // p2p wss address
	P2PWSSEnable                *bool          `yaml:"p2p-wss-enable,omitempty"`                 // enable Secure WebSocket P2P connections
	Password                    *string        `yaml:"password,omitempty"`                       // password for decrypting keys
	PasswordFile                *string        `yaml:"password-file,omitempty"`                  // path to a file that contains password for decrypting keys
	PaymentEarly                *int64         `yaml:"payment-early-percent,omitempty"`          // percentage below the peers payment threshold when we initiate settlement
	PaymentThreshold            *string        `yaml:"payment-threshold,omitempty"`              // threshold in BZZ where you expect to get paid from your peers
	PaymentTolerance            *int64         `yaml:"payment-tolerance-percent,omitempty"`      // excess debt above payment threshold in percentages where you disconnect from your peer
	PostageContractStartBlock   *uint64        `yaml:"postage-stamp-start-block,omitempty"`      // postage stamp contract start block number
	PostageStampAddress         *string        `yaml:"postage-stamp-address,omitempty"`          // postage stamp contract address
	PprofMutex                  *bool          `yaml:"pprof-mutex,omitempty"`                    // enable pprof mutex profile
	PprofProfile                *bool          `yaml:"pprof-profile,omitempty"`                  // enable pprof block profile
	PriceOracleAddress          *string        `yaml:"price-oracle-address,omitempty"`           // price oracle contract address
	RedistributionAddress       *string        `yaml:"redistribution-address,omitempty"`         // redistribution contract address
	ReserveCapacityDoubling     *int           `yaml:"reserve-capacity-doubling,omitempty"`      // reserve capacity doubling
	ResolverOptions             *[]string      `yaml:"resolver-options,omitempty"`               // ENS compatible API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url
	Resync                      *bool          `yaml:"resync,omitempty"`                         // forces the node to resync postage contract data
	SkipPostageSnapshot         *bool          `yaml:"skip-postage-snapshot,omitempty"`          // skip postage snapshot
	StakingAddress              *string        `yaml:"staking-address,omitempty"`                // staking contract address
	StatestoreCacheCapacity     *uint64        `yaml:"statestore-cache-capacity,omitempty"`      // lru memory caching capacity in number of statestore entries
	StaticNodes                 *[]string      `yaml:"static-nodes,omitempty"`                   // protect nodes from getting kicked out on bootnode
	StorageIncentivesEnable     *bool          `yaml:"storage-incentives-enable,omitempty"`      // enable storage incentives feature
	SwapEnable                  *bool          `yaml:"swap-enable,omitempty"`                    // enable swap
	SwapFactoryAddress          *string        `yaml:"swap-factory-address,omitempty"`           // swap factory addresses
	SwapInitialDeposit          *string        `yaml:"swap-initial-deposit,omitempty"`           // initial deposit if deploying a new chequebook
	TargetNeighborhood          *string        `yaml:"target-neighborhood,omitempty"`            // neighborhood to target in binary format (ex: 111111001) for mining the initial overlay
	TracingEnabled              *bool          `yaml:"tracing-enable,omitempty"`                 // enable tracing
	TracingEndpoint             *string        `yaml:"tracing-endpoint,omitempty"`               // endpoint to send tracing data
	TracingHost                 *string        `yaml:"tracing-host,omitempty"`                   // host to send tracing data
	TracingPort                 *string        `yaml:"tracing-port,omitempty"`                   // port to send tracing data
	TracingServiceName          *string        `yaml:"tracing-service-name,omitempty"`           // service name identifier for tracing
	TransactionDebugMode        *bool          `yaml:"transaction-debug-mode,omitempty"`         // skips the gas estimate step for contract transactions
	UseSIMD                     *bool          `yaml:"use-simd,omitempty"`                       // use SIMD BMT hasher (available only on linux amd64 platforms)
	Verbosity                   *string        `yaml:"verbosity,omitempty"`                      // log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace
	WarmupTime                  *time.Duration `yaml:"warmup-time,omitempty"`                    // maximum node warmup duration; proceeds when stable or after this time
	WelcomeMessage              *string        `yaml:"welcome-message,omitempty"`                // send a welcome message string during handshakes
	WithdrawAddress             *[]string      `yaml:"withdrawal-addresses-whitelist,omitempty"` // withdrawal target addresses
}

// Deref returns the value pointed to by p, or the zero value of T when p is nil.
// It is used for Beekeeper's own orchestration decisions on optional Config
// fields; it never substitutes any of Bee's defaults and is never written into
// the node's .bee.yaml.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
