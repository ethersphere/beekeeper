package config

type NodeFunder struct {
	Namespace         string
	Addresses         []string
	ChainNodeEndpoint string
	WalletKey         string // Hex encoded key
	MinAmounts        MinAmounts
}

type MinAmounts struct {
	NativeCoin float64 // on mainnet this is ETH
	SwarmToken float64 // on mainnet this is BZZ
}
