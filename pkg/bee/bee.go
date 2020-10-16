package bee

var (
	// DefaultConfig represents default Bee config
	DefaultConfig = Config{
		DataDir:              ".bee",
		DBCapacity:           5000000,
		Password:             "",
		PasswordFile:         "",
		APIAddr:              ":8080",
		P2PAddr:              ":7070",
		NATAddr:              "",
		P2PWSEnable:          false,
		P2PQUICEnable:        false,
		Bootnodes:            []string{"/dnsaddr/bootnode.ethswarm.org"},
		DebugAPIEnable:       false,
		DebugAPIAddr:         ":6060",
		NetworkID:            1,
		CORSAllowedOrigins:   []string{},
		Standalone:           false,
		TracingEnabled:       false,
		TracingEndpoint:      "127.0.0.1:6831",
		TracingServiceName:   "bee",
		Verbosity:            "info",
		WelcomeMessage:       "",
		GlobalPinningEnabled: false,
		PaymentThreshold:     100000,
		PaymentTolerance:     10000,
		PaymentEarly:         10000,
		ResolverEndpoints:    []string{},
		GatewayMode:          false,
		ClefSignerEnable:     false,
		ClefSignerEndpoint:   "",
		SwapEndpoint:         "http://localhost:8545",
		SwapFactoryAddress:   "",
		SwapInitialDeposit:   100000000,
		SwapEnable:           true,
	}
)

// Config ...
type Config struct {
	// data directory
	DataDir string
	// db capacity in chunks, multiply by 4000 (swarm.ChunkSize) to get approximate capacity in bytes
	DBCapacity uint64
	// password for decrypting keys
	Password string
	// path to a file that contains password for decrypting keys
	PasswordFile string
	// HTTP API listen address
	APIAddr string
	// P2P listen address
	P2PAddr string
	// NAT exposed address
	NATAddr string
	// enable P2P WebSocket transport
	P2PWSEnable bool
	// enable P2P QUIC transport
	P2PQUICEnable bool
	// initial nodes to connect to
	Bootnodes []string
	// enable debug HTTP API
	DebugAPIEnable bool
	// debug HTTP API listen address
	DebugAPIAddr string
	// ID of the Swarm network
	NetworkID uint64
	// origins with CORS headers enabled
	CORSAllowedOrigins []string
	// whether we want the node to start with no listen addresses for p2p
	Standalone bool
	// enable tracing
	TracingEnabled bool
	// endpoint to send tracing data
	TracingEndpoint string
	// service name identifier for tracing
	TracingServiceName string
	// log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace
	Verbosity string
	// send a welcome message string during handshakes
	WelcomeMessage string
	// enable global pinning
	GlobalPinningEnabled bool
	// threshold in BZZ where you expect to get paid from your peers
	PaymentThreshold uint64
	// excess debt above payment threshold in BZZ where you disconnect from your peer
	PaymentTolerance uint64
	// amount in BZZ below the peers payment threshold when we initiate settlement
	PaymentEarly uint64
	// ENS compatible API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url
	ResolverEndpoints []string
	// disable a set of sensitive features in the api
	GatewayMode bool
	// enable clef signer
	ClefSignerEnable bool
	// clef signer endpoint
	ClefSignerEndpoint string
	// swap ethereum blockchain endpoint
	SwapEndpoint string
	// swap factory address
	SwapFactoryAddress string
	// initial deposit if deploying a new chequebook
	SwapInitialDeposit uint64
	// enable swap
	SwapEnable bool
}
