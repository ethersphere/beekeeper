package bee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/swap"
)

// Client manages communication with the Bee node
type Client struct {
	api           *api.Client
	swapClient    swap.BlockTimeFetcher
	log           logging.Logger
	name          string
	nodeGroupName string
	apiURL        *url.URL
	retryCount    int
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	APIURL        *url.URL
	Name          string
	NodeGroupName string
	Retry         int
	SwapClient    swap.BlockTimeFetcher
	HTTPClient    *http.Client
	Logger        logging.Logger
}

// NewClient returns Bee client
func NewClient(opts ClientOptions) (c *Client, err error) {
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{}
	}

	c = &Client{
		retryCount:    5,
		log:           opts.Logger,
		swapClient:    opts.SwapClient,
		name:          opts.Name,
		nodeGroupName: opts.NodeGroupName,
		apiURL:        opts.APIURL,
	}

	c.api, err = api.NewClient(opts.APIURL, opts.HTTPClient)
	if err != nil {
		return nil, fmt.Errorf("new api client: %w", err)
	}

	if opts.Retry > 0 {
		c.retryCount = opts.Retry
	}

	return c, nil
}

// Addresses represents node's addresses
type Addresses struct {
	Overlay      swarm.Address
	Underlay     []string
	Ethereum     string
	PublicKey    string
	PSSPublicKey string
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) NodeGroup() string {
	return c.nodeGroupName
}

func (c *Client) Host() string {
	return c.apiURL.Host
}

func (c *Client) API() *api.Client {
	return c.api
}

// Addresses returns node's addresses
func (c *Client) Addresses(ctx context.Context) (resp Addresses, err error) {
	a, err := c.api.Node.Addresses(ctx)
	if err != nil {
		return Addresses{}, fmt.Errorf("get addresses: %w", err)
	}

	return Addresses{
		Ethereum:     a.Ethereum,
		Overlay:      a.Overlay,
		PublicKey:    a.PublicKey,
		Underlay:     a.Underlay,
		PSSPublicKey: a.PSSPublicKey,
	}, nil
}

// Account represents node's account with a given peer
type Account struct {
	Balance                  int64
	ConsumedBalance          int64
	GhostBalance             int64
	Peer                     string
	ReservedBalance          int64
	ShadowReservedBalance    int64
	SurplusBalance           int64
	ThresholdGiven           int64
	ThresholdReceived        int64
	CurrentThresholdGiven    int64
	CurrentThresholdReceived int64
}

// Accounting represents node's accounts with all peers
type Accounting struct {
	Accounting []Account
}

// Accounting returns node's accounts with all peers
func (c *Client) Accounting(ctx context.Context) (resp Accounting, err error) {
	r, err := c.api.Node.Accounting(ctx)
	if err != nil {
		return Accounting{}, fmt.Errorf("get accounting: %w", err)
	}

	for peer, b := range r.Accounting {
		resp.Accounting = append(resp.Accounting, Account{
			Balance:                  b.Balance.Int64(),
			ConsumedBalance:          b.ConsumedBalance.Int64(),
			ThresholdReceived:        b.ThresholdReceived.Int64(),
			ThresholdGiven:           b.ThresholdGiven.Int64(),
			SurplusBalance:           b.SurplusBalance.Int64(),
			CurrentThresholdReceived: b.CurrentThresholdReceived.Int64(),
			CurrentThresholdGiven:    b.CurrentThresholdGiven.Int64(),
			ReservedBalance:          b.ReservedBalance.Int64(),
			ShadowReservedBalance:    b.ShadowReservedBalance.Int64(),
			GhostBalance:             b.GhostBalance.Int64(),
			Peer:                     peer,
		})
	}

	return resp, err
}

// Balance represents node's balance with peer
type Balance struct {
	Balance int64
	Peer    string
}

// Balance returns node's balance with a given peer
func (c *Client) Balance(ctx context.Context, a swarm.Address) (resp Balance, err error) {
	b, err := c.api.Node.Balance(ctx, a)
	if err != nil {
		return Balance{}, fmt.Errorf("get balance with node %s: %w", a.String(), err)
	}

	return Balance{
		Balance: b.Balance.Int64(),
		Peer:    b.Peer,
	}, nil
}

// Balances represents Balances's response
type Balances struct {
	Balances []Balance
}

// Balances returns node's balances
func (c *Client) Balances(ctx context.Context) (resp Balances, err error) {
	r, err := c.api.Node.Balances(ctx)
	if err != nil {
		return Balances{}, fmt.Errorf("get balances: %w", err)
	}

	for _, b := range r.Balances {
		resp.Balances = append(resp.Balances, Balance{
			Peer:    b.Peer,
			Balance: b.Balance.Int64(),
		})
	}

	return resp, err
}

// DownloadBytes downloads chunk from the node
func (c *Client) DownloadBytes(ctx context.Context, a swarm.Address, opts *api.DownloadOptions) (data []byte, err error) {
	r, err := c.api.Bytes.Download(ctx, a, opts)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}
	defer r.Close()

	return io.ReadAll(r)
}

// DownloadChunk downloads chunk from the node
func (c *Client) DownloadChunk(ctx context.Context, a swarm.Address, targets string, opts *api.DownloadOptions) (data []byte, err error) {
	r, err := c.api.Chunks.Download(ctx, a, targets, opts)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}
	defer r.Close()

	return io.ReadAll(r)
}

// DownloadFileBytes downloads a flie from the node and returns the data.
func (c *Client) DownloadFileBytes(ctx context.Context, a swarm.Address, opts *api.DownloadOptions) (data []byte, err error) {
	r, err := c.api.Files.Download(ctx, a, opts)
	if err != nil {
		return nil, fmt.Errorf("download file %s: %w", a, err)
	}
	defer r.Close()

	return io.ReadAll(r)
}

// DownloadFile downloads chunk from the node and returns it's size and hash.
func (c *Client) DownloadFile(ctx context.Context, a swarm.Address, opts *api.DownloadOptions) (size int64, hash []byte, err error) {
	r, err := c.api.Files.Download(ctx, a, opts)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}
	defer r.Close()

	h := fileHasher()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s, hashing copy: %w", a, err)
	}

	return size, h.Sum(nil), nil
}

func (c *Client) DownloadActFile(ctx context.Context, a swarm.Address, opts *api.DownloadOptions) (size int64, hash []byte, err error) {
	r, err := c.api.Act.Download(ctx, a, opts)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}
	defer r.Close()
	h := fileHasher()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s, hashing copy: %w", a, err)
	}

	return size, h.Sum(nil), nil
}

// HasChunk returns true/false if node has a chunk
func (c *Client) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.api.Node.HasChunk(ctx, a)
}

func (c *Client) HasChunks(ctx context.Context, a []swarm.Address) (has []bool, count int, err error) {
	has = make([]bool, len(a))
	for i, addr := range a {
		v, err := c.api.Node.HasChunk(ctx, addr)
		if err != nil {
			return nil, 0, err
		}
		has[i] = v
		if v {
			count++
		}
	}
	return has, count, nil
}

// Overlay returns node's overlay address with retry mechanism
func (c *Client) Overlay(ctx context.Context) (swarm.Address, error) {
	var a api.Addresses
	var err error

	for r := 0; r < c.retryCount; r++ {
		if r > 0 {
			time.Sleep(2 * time.Duration(r) * time.Second)
		}

		a, err = c.api.Node.Addresses(ctx)
		if err == nil {
			return a.Overlay, nil
		}
	}

	return swarm.Address{}, fmt.Errorf("get addresses: %w", err)
}

// Peers returns addresses of node's peers
func (c *Client) Peers(ctx context.Context) (peers []swarm.Address, err error) {
	ps, err := c.api.Node.Peers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	for _, p := range ps.Peers {
		peers = append(peers, p.Address)
	}

	return peers, err
}

// PinRootHash pins root hash of given reference.
func (c *Client) PinRootHash(ctx context.Context, ref swarm.Address) error {
	return c.api.Pinning.PinRootHash(ctx, ref)
}

// UnpinRootHash unpins root hash of given reference.
func (c *Client) UnpinRootHash(ctx context.Context, ref swarm.Address) error {
	return c.api.Pinning.UnpinRootHash(ctx, ref)
}

// GetPinnedRootHash determines if the root hash of
// given reference is pinned by returning its reference.
func (c *Client) GetPinnedRootHash(ctx context.Context, ref swarm.Address) (swarm.Address, error) {
	return c.api.Pinning.GetPinnedRootHash(ctx, ref)
}

// GetPins returns all references of pinned root hashes.
func (c *Client) GetPins(ctx context.Context) ([]swarm.Address, error) {
	return c.api.Pinning.GetPins(ctx)
}

// Ping pings other node
func (c *Client) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := c.api.PingPong.Ping(ctx, node)
	if err != nil {
		return "", fmt.Errorf("ping node %s: %w", node, err)
	}
	return r.RTT, nil
}

// ClosestPeer returns the address of the peer closest to the node in the provided bin.
func (c *Client) ClosestPeer(ctx context.Context, binId uint8, skipList []swarm.Address) (swarm.Address, error) {
	t, err := c.Topology(ctx)
	if err != nil {
		return swarm.ZeroAddress, err
	}

	bin := t.Bins[fmt.Sprintf("bin_%d", binId)]
	if len(bin.ConnectedPeers) == 0 {
		return swarm.ZeroAddress, nil
	}

	overlay, err := c.Overlay(ctx)
	if err != nil {
		return swarm.ZeroAddress, err
	}

	closest := swarm.ZeroAddress
	minProx := uint8(math.MaxUint8)
	for _, peer := range bin.ConnectedPeers {
		skip := slices.ContainsFunc(skipList, func(addr swarm.Address) bool {
			return addr.Equal(peer.Address)
		})
		if skip {
			continue
		}
		prox := swarm.Proximity(overlay.Bytes(), peer.Address.Bytes())
		if prox < minProx {
			minProx = prox
			closest = peer.Address
		}
	}
	return closest, nil
}

// PingStreamMsg represents message sent over the PingStream channel
type PingStreamMsg struct {
	Node  swarm.Address
	RTT   string
	Index int
	Error error
}

// PingStream returns stream of ping results for given nodes
func (c *Client) PingStream(ctx context.Context, nodes []swarm.Address) <-chan PingStreamMsg {
	pingStream := make(chan PingStreamMsg)

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node swarm.Address) {
			defer wg.Done()

			rtt, err := c.Ping(ctx, node)
			pingStream <- PingStreamMsg{
				Node:  node,
				RTT:   rtt,
				Index: i,
				Error: err,
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(pingStream)
	}()

	return pingStream
}

// Settlement represents node's settlement with peer
type Settlement struct {
	Peer     string
	Received int64
	Sent     int64
}

// Settlement returns node's settlement with a given peer
func (c *Client) Settlement(ctx context.Context, a swarm.Address) (resp Settlement, err error) {
	b, err := c.api.Node.Settlement(ctx, a)
	if err != nil {
		return Settlement{}, fmt.Errorf("get settlement with node %s: %w", a.String(), err)
	}

	return Settlement{
		Peer:     b.Peer,
		Received: b.Received.Int64(),
		Sent:     b.Sent.Int64(),
	}, nil
}

// CreatePostageBatch returns the batchID of a batch of postage stamps
func (c *Client) CreatePostageBatch(ctx context.Context, amount int64, depth uint64, label string, verbose bool) (string, error) {
	if depth < MinimumBatchDepth {
		depth = MinimumBatchDepth
	}
	if verbose {
		rs, err := c.ReserveState(ctx)
		if err != nil {
			return "", fmt.Errorf("print reserve state (before): %w", err)
		}
		c.log.Infof("reserve state (prior to buying the batch):%s", rs.String())
	}
	id, err := c.api.Postage.CreatePostageBatch(ctx, amount, depth, label)
	if err != nil {
		return "", fmt.Errorf("create postage stamp: %w", err)
	}

	exists := false
	usable := false
	// wait for the stamp to become usable
	for range 900 {
		time.Sleep(1 * time.Second)
		state, err := c.api.Postage.PostageStamp(ctx, id)
		if err != nil {
			continue
		}
		exists = state.Exists
		usable = state.Usable
		if usable {
			break
		}
	}

	if !exists {
		return "", fmt.Errorf("batch %s does not exist", id)
	}

	if !usable {
		return "", fmt.Errorf("batch %s not usable withn given timeout", id)
	}

	if verbose {
		rs, err := c.ReserveState(ctx)
		if err != nil {
			return "", fmt.Errorf("print reserve state (after): %w", err)
		}
		c.log.Infof("reserve state (after buying the batch): %s", rs.String())
		c.log.Infof("created batch id %s with depth %d and amount %d", id, depth, amount)
	}
	return id, nil
}

func (c *Client) GetOrCreateMutableBatch(ctx context.Context, postageTTL time.Duration, depth uint64, label string) (string, error) {
	csr, err := c.api.Postage.GetChainState(ctx)
	if err != nil {
		return "", fmt.Errorf("get chain state: %w", err)
	}

	blockTime, err := c.swapClient.FetchBlockTime(ctx, swap.WithOffset(1000))
	if err != nil {
		return "", fmt.Errorf("fetching block time: %w", err)
	}

	amount := int64(1) // use 1 as the minimum amount when the chain price is 0, since any amount is valid

	price := csr.CurrentPrice.Int64()
	if price > 0 {
		amount = (int64(postageTTL.Seconds()) / blockTime) * price
	} else {
		c.log.Warningf("invalid chain price: %v", price)
	}

	c.log.Debugf("calculated mutable batch: amount '%d', label '%s', price '%d', block time '%d', ttl '%s'", amount, label, price, blockTime, postageTTL)

	batches, err := c.PostageBatches(ctx)
	if err != nil {
		return "", fmt.Errorf("get postage batches: %w", err)
	}

	for _, b := range batches {
		if !b.Exists {
			continue
		}
		if b.ImmutableFlag { // skip immutable batches
			continue
		}
		if b.Label != label {
			continue
		}

		if b.Usable && (b.BatchTTL == -1 || b.BatchTTL > 0) && b.Utilization < (1<<(b.Depth-b.BucketDepth)) {
			return b.BatchID, nil
		}
	}

	return c.CreatePostageBatch(ctx, amount, depth, label, false)
}

// PostageBatches returns the list of batches of node
func (c *Client) PostageBatches(ctx context.Context) ([]api.PostageStampResponse, error) {
	return c.api.Postage.PostageBatches(ctx)
}

// PostageStamp returns the batch by ID
func (c *Client) PostageStamp(ctx context.Context, batchID string) (api.PostageStampResponse, error) {
	return c.api.Postage.PostageStamp(ctx, batchID)
}

// TopUpPostageBatch tops up the given batch with the amount per chunk
func (c *Client) TopUpPostageBatch(ctx context.Context, batchID string, amount int64, gasPrice string) error {
	batch, err := c.PostageStamp(ctx, batchID)
	if err != nil {
		return fmt.Errorf("unable to retrieve batch details: %w", err)
	}

	err = c.api.Postage.TopUpPostageBatch(ctx, batchID, amount, gasPrice)
	if err != nil {
		return err
	}

	for range 60 {
		time.Sleep(time.Second)

		b, err := c.PostageStamp(ctx, batchID)
		if err != nil {
			return err
		}

		if b.Amount.Cmp(batch.Amount.Int) > 0 {
			// topup is complete
			return nil
		}
	}

	return errors.New("timed out waiting for batch topup confirmation")
}

// DilutePostageBatch dilutes the given batch by increasing the depth
func (c *Client) DilutePostageBatch(ctx context.Context, batchID string, depth uint64, gasPrice string) error {
	batch, err := c.api.Postage.PostageStamp(ctx, batchID)
	if err != nil {
		return fmt.Errorf("unable to retrieve batch details: %w", err)
	}

	err = c.api.Postage.DilutePostageBatch(ctx, batchID, depth, gasPrice)
	if err != nil {
		return err
	}

	for range 60 {
		time.Sleep(time.Second)

		b, err := c.api.Postage.PostageStamp(ctx, batchID)
		if err != nil {
			return err
		}

		if b.Depth > batch.Depth {
			// dilution is complete
			return nil
		}
	}

	return errors.New("timed out waiting for batch dilution confirmation")
}

// ReserveState returns reserve radius, available capacity, inner and outer radiuses
func (c *Client) ReserveState(ctx context.Context) (api.ReserveState, error) {
	return c.api.Postage.ReserveState(ctx)
}

// SendPSSMessage triggers a PSS message with a topic and recipient address
func (c *Client) SendPSSMessage(ctx context.Context, nodeAddress swarm.Address, publicKey string, topic string, prefix int, data []byte, batchID string) error {
	return c.api.PSS.SendMessage(ctx, nodeAddress, publicKey, topic, prefix, bytes.NewReader(data), batchID)
}

// UploadSOC uploads a single owner chunk to a node with a E
func (c *Client) UploadSOC(ctx context.Context, owner, ID, signature string, data []byte, batchID string) (swarm.Address, error) {
	resp, err := c.api.SOC.UploadSOC(ctx, owner, ID, signature, bytes.NewReader(data), batchID)
	if err != nil {
		return swarm.ZeroAddress, err
	}

	return resp.Reference, nil
}

// Settlements represents Settlements's response
type Settlements struct {
	Settlements   []Settlement
	TotalReceived int64
	TotalSent     int64
}

// Settlements returns node's settlements
func (c *Client) Settlements(ctx context.Context) (resp Settlements, err error) {
	r, err := c.api.Node.Settlements(ctx)
	if err != nil {
		return Settlements{}, fmt.Errorf("get settlements: %w", err)
	}

	for _, b := range r.Settlements {
		resp.Settlements = append(resp.Settlements, Settlement{
			Peer:     b.Peer,
			Received: b.Received.Int64(),
			Sent:     b.Sent.Int64(),
		})
	}
	resp.TotalReceived = r.TotalReceived.Int64()
	resp.TotalSent = r.TotalSent.Int64()

	return resp, err
}

type Cheque struct {
	Beneficiary string
	Chequebook  string
	Payout      *big.Int
}

type CashoutStatusResult struct {
	Recipient  string
	LastPayout *big.Int
	Bounced    bool
}

type CashoutStatusResponse struct {
	Peer            swarm.Address
	Cheque          *Cheque
	TransactionHash *string
	Result          *CashoutStatusResult
	UncashedAmount  *big.Int
}

func (c *Client) CashoutStatus(ctx context.Context, a swarm.Address) (resp CashoutStatusResponse, err error) {
	r, err := c.api.Node.CashoutStatus(ctx, a)
	if err != nil {
		return CashoutStatusResponse{}, fmt.Errorf("cashout: %w", err)
	}

	var cashoutStatusResult *CashoutStatusResult
	if r.Result != nil {
		cashoutStatusResult = &CashoutStatusResult{
			Recipient:  r.Result.Recipient,
			LastPayout: r.Result.LastPayout.Int,
			Bounced:    r.Result.Bounced,
		}
	}

	return CashoutStatusResponse{
		Peer: r.Peer,
		Cheque: &Cheque{
			Beneficiary: r.Cheque.Beneficiary,
			Chequebook:  r.Cheque.Chequebook,
			Payout:      r.Cheque.Payout.Int,
		},
		TransactionHash: r.TransactionHash,
		Result:          cashoutStatusResult,
		UncashedAmount:  r.UncashedAmount.Int,
	}, nil
}

func (c *Client) Cashout(ctx context.Context, a swarm.Address) (resp string, err error) {
	r, err := c.api.Node.Cashout(ctx, a)
	if err != nil {
		return "", fmt.Errorf("cashout: %w", err)
	}

	return r.TransactionHash, nil
}

type ChequebookBalanceResponse struct {
	TotalBalance     *big.Int
	AvailableBalance *big.Int
}

func (c *Client) ChequebookBalance(ctx context.Context) (resp ChequebookBalanceResponse, err error) {
	r, err := c.api.Node.ChequebookBalance(ctx)
	if err != nil {
		return ChequebookBalanceResponse{}, fmt.Errorf("cashout: %w", err)
	}

	return ChequebookBalanceResponse{
		TotalBalance:     r.TotalBalance.Int,
		AvailableBalance: r.AvailableBalance.Int,
	}, nil
}

// Topology represents Kademlia topology
type Topology struct {
	Overlay             swarm.Address
	Connected           int
	Population          int
	NnLowWatermark      int
	Depth               int
	Bins                map[string]Bin
	LightNodes          Bin
	Reachability        string `json:"reachability"`        // current reachability status
	NetworkAvailability string `json:"networkAvailability"` // network availability
}

// Bin represents Kademlia bin
type Bin struct {
	Population        int            `json:"population"`
	Connected         int            `json:"connected"`
	DisconnectedPeers []api.PeerInfo `json:"disconnectedPeers"`
	ConnectedPeers    []api.PeerInfo `json:"connectedPeers"`
}

// Topology returns Kademlia topology
func (c *Client) Topology(ctx context.Context) (topology Topology, err error) {
	var t api.Topology
	for r := 0; r < c.retryCount; r++ {
		if r > 0 {
			time.Sleep(2 * time.Duration(r) * time.Second)
		}

		t, err = c.api.Node.Topology(ctx)
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return Topology{}, fmt.Errorf("get topology: %w", err)
	}

	topology = Topology{
		Overlay:             t.BaseAddr,
		Connected:           t.Connected,
		Population:          t.Population,
		NnLowWatermark:      t.NnLowWatermark,
		Depth:               t.Depth,
		Bins:                make(map[string]Bin),
		Reachability:        t.Reachability,
		NetworkAvailability: t.NetworkAvailability,
	}

	for k, b := range t.Bins {
		if b.Population > 0 {
			topology.Bins[k] = Bin{
				Connected:         b.Connected,
				ConnectedPeers:    b.ConnectedPeers,
				DisconnectedPeers: b.DisconnectedPeers,
				Population:        b.Population,
			}
		}
	}

	topology.LightNodes = Bin{
		ConnectedPeers:    t.LightNodes.ConnectedPeers,
		DisconnectedPeers: t.LightNodes.DisconnectedPeers,
		Connected:         t.LightNodes.Connected,
		Population:        t.LightNodes.Population,
	}

	return topology, err
}

// Underlay returns node's underlay addresses
func (c *Client) Underlay(ctx context.Context) ([]string, error) {
	a, err := c.api.Node.Addresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("get underlay: %w", err)
	}

	return a.Underlay, nil
}

// UploadBytes uploads bytes to the node
func (c *Client) WaitSync(ctx context.Context, UId uint64) error {
	err := c.api.Tags.WaitSync(ctx, UId)
	if err != nil {
		return fmt.Errorf("sync tag: %w", err)
	}

	return err
}

// UploadBytes uploads bytes to the node
func (c *Client) UploadBytes(ctx context.Context, b []byte, o api.UploadOptions) (swarm.Address, error) {
	r, err := c.api.Bytes.Upload(ctx, bytes.NewReader(b), o)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("upload bytes: %w", err)
	}

	return r.Reference, nil
}

// UploadChunk uploads chunk to the node
func (c *Client) UploadChunk(ctx context.Context, data []byte, o api.UploadOptions) (swarm.Address, error) {
	resp, err := c.api.Chunks.Upload(ctx, data, o)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("upload chunk: %w", err)
	}

	return resp.Reference, nil
}

// UploadFile uploads file to the node
func (c *Client) UploadFile(ctx context.Context, f *File, o api.UploadOptions) (err error) {
	c.log.Debugf("uploading file %s of size %d to %s", f.Name(), f.Size(), c.apiURL.Host)

	h := fileHasher()
	start := time.Now()
	r, err := c.api.Files.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), f.Size(), o)
	duration := time.Since(start)
	if err != nil {
		c.log.Errorf("uploading file failed after %v: %v", duration, err)
		return fmt.Errorf("upload file: %w", err)
	}

	f.SetAddress(r.Reference)
	f.SetHash(h.Sum(nil))

	c.log.Debugf("successfully uploaded file %s in %v", f.Name(), duration)

	return nil
}

func (c *Client) UploadActFile(ctx context.Context, f *File, o api.UploadOptions) (err error) {
	h := fileHasher()
	r, err := c.api.Act.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), o)
	if err != nil {
		return fmt.Errorf("upload ACT file: %w", err)
	}

	f.SetAddress(r.Reference)
	f.SetHistroryAddress(r.HistoryAddress)
	f.SetHash(h.Sum(nil))

	return nil
}

func (c *Client) AddActGrantees(ctx context.Context, f *File, o api.UploadOptions) (err error) {
	h := fileHasher()
	r, err := c.api.Act.AddGrantees(ctx, io.TeeReader(f.DataReader(), h), o)
	if err != nil {
		return fmt.Errorf("add ACT grantees: %w", err)
	}

	f.SetAddress(r.Reference)
	f.SetHistroryAddress(r.HistoryAddress)
	f.SetHash(h.Sum(nil))

	return nil
}

func (c *Client) GetActGrantees(ctx context.Context, a swarm.Address) (addresses []string, err error) {
	r, e := c.api.Act.GetGrantees(ctx, a)
	if e != nil {
		return nil, fmt.Errorf("get grantees: %s: %w", a, e)
	}
	defer r.Close()
	err = json.NewDecoder(r).Decode(&addresses)
	return addresses, err
}

func (c *Client) PatchActGrantees(ctx context.Context, pf *File, addr swarm.Address, haddr swarm.Address, batchID string) (err error) {
	r, err := c.api.Act.PatchGrantees(ctx, pf.DataReader(), addr, haddr, batchID)
	if err != nil {
		return fmt.Errorf("add ACT grantees: %w", err)
	}

	pf.SetAddress(r.Reference)
	pf.SetHistroryAddress(r.HistoryAddress)
	return nil
}

// UploadCollection uploads TAR collection bytes to the node
func (c *Client) UploadCollection(ctx context.Context, f *File, o api.UploadOptions) (err error) {
	h := fileHasher()
	r, err := c.api.Dirs.Upload(ctx, io.TeeReader(f.DataReader(), h), f.Size(), o)
	if err != nil {
		return fmt.Errorf("upload collection: %w", err)
	}

	f.SetAddress(r.Reference)
	f.SetHash(h.Sum(nil))

	return err
}

// DownloadManifestFile downloads manifest file from the node and returns it's size and hash
func (c *Client) DownloadManifestFile(ctx context.Context, a swarm.Address, path string) (size int64, hash []byte, err error) {
	r, err := c.api.Dirs.Download(ctx, a, path)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file `%s`: %w", path, err)
	}
	defer r.Close()

	h := fileHasher()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file `%s`: %w", path, err)
	}

	return size, h.Sum(nil), nil
}

// CreateTag creates tag on the node
func (c *Client) CreateTag(ctx context.Context) (resp api.TagResponse, err error) {
	resp, err = c.api.Tags.CreateTag(ctx)
	if err != nil {
		return api.TagResponse{}, fmt.Errorf("create tag: %w", err)
	}

	return resp, nil
}

// GetTag retrieves tag from node
func (c *Client) GetTag(ctx context.Context, tagUID uint64) (resp api.TagResponse, err error) {
	resp, err = c.api.Tags.GetTag(ctx, tagUID)
	if err != nil {
		return api.TagResponse{}, fmt.Errorf("get tag: %w", err)
	}

	return resp, nil
}

// IsRetrievable checks whether the content on the given address is retrievable.
func (c *Client) IsRetrievable(ctx context.Context, ref swarm.Address) (bool, error) {
	return c.api.Stewardship.IsRetrievable(ctx, ref)
}

// Reupload re-uploads root hash and all of its underlying associated chunks to
// the network.
func (c *Client) Reupload(ctx context.Context, ref swarm.Address) error {
	return c.api.Stewardship.Reupload(ctx, ref)
}

// DepositStake deposits stake
func (c *Client) DepositStake(ctx context.Context, amount *big.Int) (string, error) {
	return c.api.Stake.DepositStake(ctx, amount)
}

// GetStake returns stake amount
func (c *Client) GetStake(ctx context.Context) (*big.Int, error) {
	return c.api.Stake.GetStakedAmount(ctx)
}

// GetWithdrawableStake returns withdrawable amount
func (c *Client) GetWithdrawableStake(ctx context.Context) (*big.Int, error) {
	return c.api.Stake.GetWithdrawableStake(ctx)
}

// MigrateStake withdraws stake
func (c *Client) MigrateStake(ctx context.Context) (string, error) {
	return c.api.Stake.MigrateStake(ctx)
}

// WalletBalance fetches the balance for the given token
func (c *Client) WalletBalance(ctx context.Context, token string) (*big.Int, error) {
	resp, err := c.api.Node.Wallet(ctx)
	if err != nil {
		return nil, err
	}

	if token == "BZZ" {
		return resp.BZZ.Int, nil
	}

	return resp.NativeToken.Int, nil
}

// Withdraw transfers token from eth address to the provided address
func (c *Client) Withdraw(ctx context.Context, token, addr string, amount int64) error {
	resp, err := c.api.Node.Withdraw(ctx, token, addr, amount)
	if err != nil {
		return err
	}

	var zeroHash common.Hash

	if resp == zeroHash {
		return errors.New("withdraw returned zero hash")
	}

	return nil
}

// CreateRootFeedManifest creates an initial root manifest
func (c *Client) CreateRootFeedManifest(ctx context.Context, signer crypto.Signer, topic []byte, o api.UploadOptions) (*api.FeedUploadResponse, error) {
	return c.api.Feed.CreateRootManifest(ctx, signer, topic, o)
}

// UpdateFeedWithRootChunk updates a feed with a root chunk
func (c *Client) UpdateFeedWithRootChunk(ctx context.Context, signer crypto.Signer, topic []byte, i uint64, ch swarm.Chunk, o api.UploadOptions) (*api.SocResponse, error) {
	return c.api.Feed.UpdateWithRootChunk(ctx, signer, topic, i, ch, o)
}

// FindFeedUpdate finds the latest update for a feed
func (c *Client) FindFeedUpdate(ctx context.Context, signer crypto.Signer, topic []byte, o *api.DownloadOptions) (*api.FindFeedUpdateResponse, error) {
	return c.api.Feed.FindUpdate(ctx, signer, topic, o)
}

func (c *Client) Status(ctx context.Context) (*api.StatusResponse, error) {
	return c.api.Status.Status(ctx)
}
