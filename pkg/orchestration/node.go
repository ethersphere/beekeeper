package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

// ErrNotSet represents error when orchestration client is not set
var ErrNotSet = errors.New("orchestration client not set")

type Node interface {
	Client() *bee.Client
	Config() *Config
	LibP2PKey() string
	Name() string
	SetSwarmKey(key string) Node
	SwarmKey() string
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

// EncryptedKey is part of Ethereum JSON v3 key file format.
type EncryptedKey string

func (ek EncryptedKey) String() string {
	return string(ek)
}

// EncryptedKeyJson is json string for EncryptedKey.
type EncryptedKeyJson struct {
	Address string `json:"address"`
	// TODO map complete key to Ethereum JSON v3 key file format
}

// GetEthAddress extracts ethereum address from EncryptedKey.
func (ek EncryptedKey) GetEthAddress() (string, error) {
	var skj EncryptedKeyJson

	err := json.Unmarshal([]byte(ek), &skj)
	if err != nil {
		return "", fmt.Errorf("unmarshal swarm encrypted key address: %s", err.Error())
	}

	if skj.Address != "" && !strings.HasPrefix(skj.Address, "0x") {
		skj.Address = fmt.Sprintf("0x%s", skj.Address)
	}

	return skj.Address, nil
}

// NodeOptions holds optional parameters for the Node.
type NodeOptions struct {
	Client    *bee.Client
	Config    *Config
	LibP2PKey string
	SwarmKey  EncryptedKey
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
	AllowPrivateCIDRs         bool          // allow to advertise private CIDRs to the public network
	APIAddr                   string        // HTTP API listen address
	BlockTime                 uint64        // chain block time
	Bootnodes                 string        // initial nodes to connect to
	BootnodeMode              bool          // cause the node to always accept incoming connections
	CacheCapacity             uint64        // cache capacity in chunks, multiply by 4096 (MaxChunkSize) to get approximate capacity in bytes
	CORSAllowedOrigins        string        // origins with CORS headers enabled
	DataDir                   string        // data directory
	DbOpenFilesLimit          int           // number of open files allowed by database
	DbBlockCacheCapacity      int           // size of block cache of the database in bytes
	DbWriteBufferSize         int           // size of the database write buffer in bytes
	DbDisableSeeksCompaction  bool          // disables DB compactions triggered by seeks
	FullNode                  bool          // cause the node to start in full mode
	Mainnet                   bool          // enable mainnet
	NATAddr                   string        // NAT exposed address
	NetworkID                 uint64        // ID of the Swarm network
	P2PAddr                   string        // P2P listen address
	P2PWSEnable               bool          // enable P2P WebSocket transport
	Password                  string        // password for decrypting keys
	PaymentEarly              uint64        // amount in BZZ below the peers payment threshold when we initiate settlement
	PaymentThreshold          uint64        // threshold in BZZ where you expect to get paid from your peers
	PaymentTolerance          uint64        // excess debt above payment threshold in BZZ where you disconnect from your peer
	PostageStampAddress       string        // postage stamp address
	PostageContractStartBlock uint64        // postage stamp address
	PriceOracleAddress        string        // price Oracle address
	ResolverOptions           string        // ENS compatible API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url
	ChequebookEnable          bool          // enable chequebook
	SwapEnable                bool          // enable swap
	SwapEndpoint              string        // swap ethereum blockchain endpoint
	SwapDeploymentGasPrice    string        // gas price in wei to use for deployment and funding
	SwapFactoryAddress        string        // swap factory address
	RedistributionAddress     string        // redistribution address
	StakingAddress            string        // staking address
	StorageIncentivesEnable   string        // storage incentives enable flag
	SwapInitialDeposit        uint64        // initial deposit if deploying a new chequebook
	TracingEnabled            bool          // enable tracing
	TracingEndpoint           string        // endpoint to send tracing data
	TracingServiceName        string        // service name identifier for tracing
	Verbosity                 uint64        // log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace
	WelcomeMessage            string        // send a welcome message string during handshakes
	WarmupTime                time.Duration // warmup time pull/pushsync protocols
	WithdrawAddress           string        // allowed addresses for wallet withdrawal
}
