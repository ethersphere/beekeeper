package config

import (
	"reflect"

	"github.com/ethersphere/beekeeper/pkg/k8s"
)

// BeeConfig represents Bee configuration
type BeeConfig struct {
	// parent to inherit settings from
	*Inherit `yaml:",inline"`
	// Bee configuration
	APIAddr                    *string `yaml:"api-addr"`
	BlockTime                  *uint64 `yaml:"block-time"`
	Bootnodes                  *string `yaml:"bootnodes"`
	ClefSignerEnable           *bool   `yaml:"clef-signer-enable"`
	ClefSignerEndpoint         *string `yaml:"clef-signer-endpoint"`
	CORSAllowedOrigins         *string `yaml:"cors-allowed-origins"`
	DataDir                    *string `yaml:"data-dir"`
	DBCapacity                 *uint64 `yaml:"db-capacity"`
	DebugAPIAddr               *string `yaml:"debug-api-addr"`
	DebugAPIEnable             *bool   `yaml:"debug-api-enable"`
	FullNode                   *bool   `yaml:"full-node"`
	GatewayMode                *bool   `yaml:"gateway-mode"`
	GlobalPinningEnabled       *bool   `yaml:"global-pinning-enabled"`
	NATAddr                    *string `yaml:"nat-addr"`
	NetworkID                  *uint64 `yaml:"network-id"`
	P2PAddr                    *string `yaml:"p2p-addr"`
	P2PQUICEnable              *bool   `yaml:"p2p-quic-enable"`
	P2PWSEnable                *bool   `yaml:"pwp-ws-enable"`
	Password                   *string `yaml:"password"`
	PaymentEarly               *uint64 `yaml:"payment-early"`
	PaymentThreshold           *uint64 `yaml:"payment-threshold"`
	PaymentTolerance           *uint64 `yaml:"payment-tolerance"`
	PostageStampAddress        *string `yaml:"postage-stamp-address"`
	PriceOracleAddress         *string `yaml:"price-oracle-address"`
	ResolverOptions            *string `yaml:"resolver-options"`
	Standalone                 *bool   `yaml:"standalone"`
	SwapEnable                 *bool   `yaml:"swap-enable"`
	SwapEndpoint               *string `yaml:"swap-endpoint"`
	SwapFactoryAddress         *string `yaml:"swap-factory-address"`
	SwapLegacyFactoryAddresses *string `yaml:"swap-legacy-factory-addresses"`
	SwapInitialDeposit         *uint64 `yaml:"swap-initial-deposit"`
	TracingEnabled             *bool   `yaml:"tracing-enabled"`
	TracingEndpoint            *string `yaml:"tracing-endpoint"`
	TracingServiceName         *string `yaml:"tracing-service-name"`
	Verbosity                  *uint64 `yaml:"verbosity"`
	WelcomeMessage             *string `yaml:"welcome-message"`
}

// Export exports BeeConfig to k8s.Config
func (b *BeeConfig) Export() (o k8s.Config) {
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

	return remoteVal.Interface().(k8s.Config)
}
