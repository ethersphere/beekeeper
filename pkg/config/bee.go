package config

import (
	"reflect"
	"time"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

type Inheritable interface {
	GetParentName() string
}

// BeeConfig represents Bee configuration
type BeeConfig struct {
	// parent to inherit settings from
	*Inherit `yaml:",inline"`
	// Bee configuration
	AllowPrivateCIDRs         *bool          `yaml:"allow-private-cidrs"`
	APIAddr                   *string        `yaml:"api-addr"`
	BlockchainRPCEndpoint     *string        `yaml:"blockchain-rpc-endpoint"`
	BlockTime                 *uint64        `yaml:"block-time"`
	BootnodeMode              *bool          `yaml:"bootnode-mode"`
	Bootnodes                 *string        `yaml:"bootnodes"`
	CacheCapacity             *uint64        `yaml:"cache-capacity"`
	ChequebookEnable          *bool          `yaml:"chequebook-enable"`
	CORSAllowedOrigins        *string        `yaml:"cors-allowed-origins"`
	DataDir                   *string        `yaml:"data-dir"`
	DbBlockCacheCapacity      *int           `yaml:"db-block-cache-capacity"`
	DbDisableSeeksCompaction  *bool          `yaml:"db-disable-seeks-compaction"`
	DbOpenFilesLimit          *int           `yaml:"db-open-files-limit"`
	DbWriteBufferSize         *int           `yaml:"db-write-buffer-size"`
	FullNode                  *bool          `yaml:"full-node"`
	Mainnet                   *bool          `yaml:"mainnet"`
	NATAddr                   *string        `yaml:"nat-addr"`
	NetworkID                 *uint64        `yaml:"network-id"`
	P2PAddr                   *string        `yaml:"p2p-addr"`
	P2PWSEnable               *bool          `yaml:"p2p-ws-enable"`
	Password                  *string        `yaml:"password"`
	PaymentEarly              *uint64        `yaml:"payment-early-percent"`
	PaymentThreshold          *uint64        `yaml:"payment-threshold"`
	PaymentTolerance          *uint64        `yaml:"payment-tolerance-percent"`
	PostageContractStartBlock *uint64        `yaml:"postage-stamp-start-block"`
	PostageStampAddress       *string        `yaml:"postage-stamp-address"`
	PriceOracleAddress        *string        `yaml:"price-oracle-address"`
	RedistributionAddress     *string        `yaml:"redistribution-address"`
	ResolverOptions           *string        `yaml:"resolver-options"`
	StakingAddress            *string        `yaml:"staking-address"`
	StorageIncentivesEnable   *string        `yaml:"storage-incentives-enable"`
	SwapEnable                *bool          `yaml:"swap-enable"`
	SwapEndpoint              *string        `yaml:"swap-endpoint"` // deprecated: use blockchain-rpc-endpoint
	SwapFactoryAddress        *string        `yaml:"swap-factory-address"`
	SwapInitialDeposit        *uint64        `yaml:"swap-initial-deposit"`
	TracingEnabled            *bool          `yaml:"tracing-enabled"`
	TracingEndpoint           *string        `yaml:"tracing-endpoint"`
	TracingServiceName        *string        `yaml:"tracing-service-name"`
	Verbosity                 *uint64        `yaml:"verbosity"`
	WarmupTime                *time.Duration `yaml:"warmup-time"`
	WelcomeMessage            *string        `yaml:"welcome-message"`
	WithdrawAddress           *string        `yaml:"withdrawal-addresses-whitelist"`
}

func (b BeeConfig) GetParentName() string {
	if b.Inherit != nil {
		return b.Inherit.ParentName
	}
	return ""
}

// Export exports BeeConfig to orchestration.Config
func (b *BeeConfig) Export() (config orchestration.Config) {
	localVal := reflect.ValueOf(b).Elem()
	localType := reflect.TypeOf(b).Elem()
	remoteVal := reflect.ValueOf(&config).Elem()

	for i := range localVal.NumField() {
		localField := localVal.Field(i)
		if localField.IsValid() && !localField.IsNil() {
			localFieldVal := localVal.Field(i).Elem()
			localFieldName := localType.Field(i).Name

			remoteFieldVal := remoteVal.FieldByName(localFieldName)
			if remoteFieldVal.IsValid() && remoteFieldVal.Type() == localFieldVal.Type() {
				remoteFieldVal.Set(localFieldVal)
			}
		}
	}

	config = remoteVal.Interface().(orchestration.Config)

	if config.BlockchainRPCEndpoint == "" && b.SwapEndpoint != nil {
		config.BlockchainRPCEndpoint = *b.SwapEndpoint
	}

	return config
}
