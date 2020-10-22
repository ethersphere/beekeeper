package bee

// Config represents Bee configuration
type Config struct {
	APIAddr              string // HTTP API listen address
	Bootnodes            string // initial nodes to connect to
	ClefSignerEnable     bool   // enable clef signer
	ClefSignerEndpoint   string // clef signer endpoint
	CORSAllowedOrigins   string // origins with CORS headers enabled
	DataDir              string // data directory
	DBCapacity           uint64 // db capacity in chunks, multiply by 4096 (MaxChunkSize) to get approximate capacity in bytes
	DebugAPIAddr         string // debug HTTP API listen address
	DebugAPIEnable       bool   // enable debug HTTP API
	GatewayMode          bool   // disable a set of sensitive features in the api
	GlobalPinningEnabled bool   // enable global pinning
	NATAddr              string // NAT exposed address
	NetworkID            uint64 // ID of the Swarm network
	P2PAddr              string // P2P listen address
	P2PQUICEnable        bool   // enable P2P QUIC transport
	P2PWSEnable          bool   // enable P2P WebSocket transport
	Password             string // password for decrypting keys
	PasswordFile         string // path to a file that contains password for decrypting keys
	PaymentEarly         uint64 // amount in BZZ below the peers payment threshold when we initiate settlement
	PaymentThreshold     uint64 // threshold in BZZ where you expect to get paid from your peers
	PaymentTolerance     uint64 // excess debt above payment threshold in BZZ where you disconnect from your peer
	ResolverEndpoints    string // ENS compatible API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url
	Standalone           bool   // whether we want the node to start with no listen addresses for p2p
	SwapEnable           bool   // enable swap
	SwapEndpoint         string // swap ethereum blockchain endpoint
	SwapFactoryAddress   string // swap factory address
	SwapInitialDeposit   uint64 // initial deposit if deploying a new chequebook
	TracingEnabled       bool   // enable tracing
	TracingEndpoint      string // endpoint to send tracing data
	TracingServiceName   string // service name identifier for tracing
	Verbosity            uint64 // log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace
	WelcomeMessage       string // send a welcome message string during handshakes
}

const configTemplate = `api-addr: {{.APIAddr}}
bootnode: {{.Bootnodes}}
clef-signer-enable: {{.ClefSignerEnable}}
clef-signer-endpoint: {{.ClefSignerEndpoint}}
cors-allowed-origins: {{.CORSAllowedOrigins}}
data-dir: {{.DataDir}}
db-capacity: {{.DBCapacity}}
debug-api-addr: {{.DebugAPIAddr}}
debug-api-enable: {{.DebugAPIEnable}}
gateway-mode: {{.GatewayMode}}
global-pinning-enable: {{.GlobalPinningEnabled}}
nat-addr: {{.NATAddr}}
network-id: {{.NetworkID}}
p2p-addr: {{.P2PAddr}}
p2p-quic-enable: {{.P2PQUICEnable}}
p2p-ws-enable: {{.P2PWSEnable}}
password: {{.Password}}
password-file: {{.PasswordFile}}
payment-early: {{.PaymentEarly}}
payment-threshold: {{.PaymentThreshold}}
payment-tolerance: {{.PaymentTolerance}}
resolver-endpoints: {{.ResolverEndpoints}}
standalone: {{.Standalone}}
swap-enable: {{.SwapEnable}}
swap-endpoint: {{.SwapEndpoint}}
swap-factory-address: {{.SwapFactoryAddress}}
swap-initial-deposit: {{.SwapInitialDeposit}}
tracing-enable: {{.TracingEnabled}}
tracing-endpoint: {{.TracingEndpoint}}
tracing-service-name: {{.TracingServiceName}}
verbosity: {{.Verbosity}}
welcome-message: {{.WelcomeMessage}}
`
