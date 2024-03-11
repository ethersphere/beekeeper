package config

import (
	"reflect"
	"time"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// BeeConfig represents Bee configuration
type BeeConfig struct {
	// parent to inherit settings from
	*Inherit `yaml:",inline"`
	// Bee configuration
	AllowPrivateCIDRs         *bool          `yaml:"allow-private-cidrs"`
	APIAddr                   *string        `yaml:"api-addr"`
	BlockTime                 *uint64        `yaml:"block-time"`
	Bootnodes                 *string        `yaml:"bootnodes"`
	BootnodeMode              *bool          `yaml:"bootnode-mode"`
	CacheCapacity             *uint64        `yaml:"cache-capacity"`
	ClefSignerEnable          *bool          `yaml:"clef-signer-enable"`
	ClefSignerEndpoint        *string        `yaml:"clef-signer-endpoint"`
	CORSAllowedOrigins        *string        `yaml:"cors-allowed-origins"`
	DataDir                   *string        `yaml:"data-dir"`
	DbOpenFilesLimit          *int           `yaml:"db-open-files-limit"`
	DbBlockCacheCapacity      *int           `yaml:"db-block-cache-capacity"`
	DbWriteBufferSize         *int           `yaml:"db-write-buffer-size"`
	DbDisableSeeksCompaction  *bool          `yaml:"db-disable-seeks-compaction"`
	DebugAPIAddr              *string        `yaml:"debug-api-addr"`
	DebugAPIEnable            *bool          `yaml:"debug-api-enable"`
	FullNode                  *bool          `yaml:"full-node"`
	NATAddr                   *string        `yaml:"nat-addr"`
	Mainnet                   *bool          `yaml:"mainnet"`
	NetworkID                 *uint64        `yaml:"network-id"`
	P2PAddr                   *string        `yaml:"p2p-addr"`
	P2PWSEnable               *bool          `yaml:"pwp-ws-enable"`
	Password                  *string        `yaml:"password"`
	PaymentEarly              *uint64        `yaml:"payment-early-percent"`
	PaymentThreshold          *uint64        `yaml:"payment-threshold"`
	PaymentTolerance          *uint64        `yaml:"payment-tolerance-percent"`
	PostageStampAddress       *string        `yaml:"postage-stamp-address"`
	PostageContractStartBlock *uint64        `yaml:"postage-stamp-start-block"`
	PriceOracleAddress        *string        `yaml:"price-oracle-address"`
	RedistributionAddress     *string        `yaml:"redistribution-address"`
	StakingAddress            *string        `yaml:"staking-address"`
	StorageIncentivesEnable   *string        `yaml:"storage-incentives-enable"`
	ResolverOptions           *string        `yaml:"resolver-options"`
	Restricted                *bool          `yaml:"restricted"`
	TokenEncryptionKey        *string        `yaml:"token-encryption-key"`
	AdminPassword             *string        `yaml:"admin-password"`
	ChequebookEnable          *bool          `yaml:"chequebook-enable"`
	SwapEnable                *bool          `yaml:"swap-enable"`
	SwapEndpoint              *string        `yaml:"swap-endpoint"`
	SwapDeploymentGasPrice    *string        `yaml:"swap-deployment-gas-price"`
	SwapFactoryAddress        *string        `yaml:"swap-factory-address"`
	SwapInitialDeposit        *uint64        `yaml:"swap-initial-deposit"`
	TracingEnabled            *bool          `yaml:"tracing-enabled"`
	TracingEndpoint           *string        `yaml:"tracing-endpoint"`
	TracingServiceName        *string        `yaml:"tracing-service-name"`
	Verbosity                 *uint64        `yaml:"verbosity"`
	WelcomeMessage            *string        `yaml:"welcome-message"`
	WarmupTime                *time.Duration `yaml:"warmup-time"`
	WithdrawAddress           *string        `yaml:"withdraw-address"`
}

// Export exports BeeConfig to orchestration.Config
func (b *BeeConfig) Export() (o orchestration.Config) {
	localVal := reflect.ValueOf(b).Elem()
	localType := reflect.TypeOf(b).Elem()
	remoteVal := reflect.ValueOf(&o).Elem()

	for i := 0; i < localVal.NumField(); i++ {
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

	return remoteVal.Interface().(orchestration.Config)
}
