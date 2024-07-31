package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/beekeeper/pkg/bigint"

	"github.com/ethersphere/bee/pkg/swarm"
)

// NodeService represents Bee's Node service
type NodeService service

// Addresses represents node's addresses
type Addresses struct {
	Ethereum     string        `json:"ethereum"`
	Overlay      swarm.Address `json:"overlay"`
	PublicKey    string        `json:"publicKey"`
	Underlay     []string      `json:"underlay"`
	PSSPublicKey string        `json:"pssPublicKey"`
}

// Addresses returns node's addresses
func (n *NodeService) Addresses(ctx context.Context) (resp Addresses, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/addresses", nil, &resp)
	return
}

// Account represents node's account with a given peer
type Account struct {
	Balance                  *bigint.BigInt `json:"balance"`
	ConsumedBalance          *bigint.BigInt `json:"consumedBalance"`
	GhostBalance             *bigint.BigInt `json:"ghostBalance"`
	ReservedBalance          *bigint.BigInt `json:"reservedBalance"`
	ShadowReservedBalance    *bigint.BigInt `json:"shadowReservedBalance"`
	SurplusBalance           *bigint.BigInt `json:"surplusBalance"`
	ThresholdReceived        *bigint.BigInt `json:"thresholdReceived"`
	ThresholdGiven           *bigint.BigInt `json:"thresholdGiven"`
	CurrentThresholdReceived *bigint.BigInt `json:"currentThresholdReceived"`
	CurrentThresholdGiven    *bigint.BigInt `json:"currentThresholdGiven"`
}

// Accounting represents node's accounts with all peers
type Accounting struct {
	Accounting map[string]Account `json:"peerData"`
}

// Accounting returns node's accounts with all peers
func (n *NodeService) Accounting(ctx context.Context) (resp Accounting, err error) {
	err = n.client.request(ctx, http.MethodGet, "/accounting", nil, &resp)
	return
}

// Balance represents node's balance with a peer
type Balance struct {
	Balance *bigint.BigInt `json:"balance"`
	Peer    string         `json:"peer"`
}

// Balance returns node's balance with a given peer
func (n *NodeService) Balance(ctx context.Context, a swarm.Address) (resp Balance, err error) {
	err = n.client.request(ctx, http.MethodGet, "/balances/"+a.String(), nil, &resp)
	return
}

// Balances represents node's balances with all peers
type Balances struct {
	Balances []Balance `json:"balances"`
}

// Balances returns node's balances with all peers
func (n *NodeService) Balances(ctx context.Context) (resp Balances, err error) {
	err = n.client.request(ctx, http.MethodGet, "/balances", nil, &resp)
	return
}

// HasChunk returns true/false if node has a chunk
func (n *NodeService) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	resp := struct {
		Message string `json:"message,omitempty"`
		Code    int    `json:"code,omitempty"`
	}{}

	err := n.client.requestJSON(ctx, http.MethodGet, "/chunks/"+a.String(), nil, &resp)
	if IsHTTPStatusErrorCode(err, http.StatusNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// Health represents node's health
type Health struct {
	Status string `json:"status"`
}

// Health returns node's health
func (n *NodeService) Health(ctx context.Context) (resp Health, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/health", nil, &resp)
	return
}

// Peers represents node's peers
type Peers struct {
	Peers []Peer `json:"peers"`
}

// Peer represents node's peer
type Peer struct {
	Address swarm.Address `json:"address"`
}

// Peers returns node's peers
func (n *NodeService) Peers(ctx context.Context) (resp Peers, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/peers", nil, &resp)
	return
}

// Readiness represents node's readiness
type Readiness struct {
	Status string `json:"status"`
}

// Readiness returns node's readiness
func (n *NodeService) Readiness(ctx context.Context) (resp Readiness, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/readiness", nil, &resp)
	return
}

// Settlement represents node's settlement with a peer
type Settlement struct {
	Peer     string         `json:"peer"`
	Received *bigint.BigInt `json:"received"`
	Sent     *bigint.BigInt `json:"sent"`
}

// Settlement returns node's settlement with a given peer
func (n *NodeService) Settlement(ctx context.Context, a swarm.Address) (resp Settlement, err error) {
	err = n.client.request(ctx, http.MethodGet, "/settlements/"+a.String(), nil, &resp)
	return
}

// Settlements represents node's settlements with all peers
type Settlements struct {
	Settlements   []Settlement   `json:"settlements"`
	TotalReceived *bigint.BigInt `json:"totalReceived"`
	TotalSent     *bigint.BigInt `json:"totalSent"`
}

// Settlements returns node's settlements with all peers
func (n *NodeService) Settlements(ctx context.Context) (resp Settlements, err error) {
	err = n.client.request(ctx, http.MethodGet, "/settlements", nil, &resp)
	return
}

type Cheque struct {
	Beneficiary string         `json:"beneficiary"`
	Chequebook  string         `json:"chequebook"`
	Payout      *bigint.BigInt `json:"payout"`
}

type CashoutStatusResult struct {
	Recipient  string         `json:"recipient"`
	LastPayout *bigint.BigInt `json:"lastPayout"`
	Bounced    bool           `json:"bounced"`
}

type CashoutStatusResponse struct {
	Peer            swarm.Address        `json:"peer"`
	Cheque          *Cheque              `json:"lastCashedCheque"`
	TransactionHash *string              `json:"transactionHash"`
	Result          *CashoutStatusResult `json:"result"`
	UncashedAmount  *bigint.BigInt       `json:"uncashedAmount"`
}

func (n *NodeService) CashoutStatus(ctx context.Context, a swarm.Address) (resp CashoutStatusResponse, err error) {
	err = n.client.request(ctx, http.MethodGet, "/chequebook/cashout/"+a.String(), nil, &resp)
	return
}

type TransactionHashResponse struct {
	TransactionHash string `json:"transactionHash"`
}

func (n *NodeService) Cashout(ctx context.Context, a swarm.Address) (resp TransactionHashResponse, err error) {
	err = n.client.request(ctx, http.MethodPost, "/chequebook/cashout/"+a.String(), nil, &resp)
	return
}

type ChequebookBalanceResponse struct {
	TotalBalance     *bigint.BigInt `json:"totalBalance"`
	AvailableBalance *bigint.BigInt `json:"availableBalance"`
}

func (n *NodeService) ChequebookBalance(ctx context.Context) (resp ChequebookBalanceResponse, err error) {
	err = n.client.request(ctx, http.MethodGet, "/chequebook/balance", nil, &resp)
	return
}

// Topology represents Kademlia topology
type Topology struct {
	BaseAddr            swarm.Address  `json:"baseAddr"`
	Population          int            `json:"population"`
	Connected           int            `json:"connected"`
	Timestamp           time.Time      `json:"timestamp"`
	NnLowWatermark      int            `json:"nnLowWatermark"`
	Depth               int            `json:"depth"`
	Bins                map[string]Bin `json:"bins"`
	LightNodes          Bin            `json:"lightNodes"`
	Reachability        string         `json:"reachability"`        // current reachability status
	NetworkAvailability string         `json:"networkAvailability"` // network availability
}

// Bin represents Kademlia bin
type Bin struct {
	Population        int        `json:"population"`
	Connected         int        `json:"connected"`
	DisconnectedPeers []PeerInfo `json:"disconnectedPeers"`
	ConnectedPeers    []PeerInfo `json:"connectedPeers"`
}

// PeerInfo is a view of peer information exposed to a user.
type PeerInfo struct {
	Address swarm.Address       `json:"address"`
	Metrics *MetricSnapshotView `json:"metrics,omitempty"`
}

// MetricSnapshotView represents snapshot of metrics counters in more human readable form.
type MetricSnapshotView struct {
	LastSeenTimestamp          int64   `json:"lastSeenTimestamp"`
	SessionConnectionRetry     uint64  `json:"sessionConnectionRetry"`
	ConnectionTotalDuration    float64 `json:"connectionTotalDuration"`
	SessionConnectionDuration  float64 `json:"sessionConnectionDuration"`
	SessionConnectionDirection string  `json:"sessionConnectionDirection"`
	LatencyEWMA                int64   `json:"latencyEWMA"`
	Reachability               string  `json:"reachability"`
}

// Topology returns Kademlia topology
func (n *NodeService) Topology(ctx context.Context) (resp Topology, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/topology", nil, &resp)
	if err != nil {
		return Topology{}, err
	}

	return
}

type Wallet struct {
	BZZ         *bigint.BigInt `json:"bzzBalance"`
	NativeToken *bigint.BigInt `json:"nativeTokenBalance"`
}

// Wallet returns the wallet state
func (n *NodeService) Wallet(ctx context.Context) (resp Wallet, err error) {
	err = n.client.requestJSON(ctx, http.MethodGet, "/wallet", nil, &resp)
	return
}

// Withdraw calls wallet withdraw endpoint
func (n *NodeService) Withdraw(ctx context.Context, token, addr string, amount int64) (tx common.Hash, err error) {
	endpoint := fmt.Sprintf("/wallet/withdraw/%s?address=%s&amount=%d", token, addr, amount)

	r := struct {
		TransactionHash common.Hash `json:"transactionHash"`
	}{}

	if err = n.client.requestJSON(ctx, http.MethodPost, endpoint, nil, &r); err != nil {
		return
	}

	return r.TransactionHash, nil
}
