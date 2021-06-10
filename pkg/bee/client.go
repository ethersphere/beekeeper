package bee

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
)

const retryCount int = 5

// Client manages communication with the Bee node
type Client struct {
	api   *api.Client
	debug *debugapi.Client
	opts  ClientOptions

	// number of times to retry call
	retry int
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	APIURL              *url.URL
	APIInsecureTLS      bool
	DebugAPIURL         *url.URL
	DebugAPIInsecureTLS bool
	Retry               int
}

// NewClient returns Bee client
func NewClient(opts ClientOptions) (c *Client) {
	c = &Client{
		retry: retryCount,
		opts:  opts,
	}

	if opts.APIURL != nil {
		c.api = api.NewClient(opts.APIURL, &api.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.APIInsecureTLS},
		}}})
	}
	if opts.DebugAPIURL != nil {
		c.debug = debugapi.NewClient(opts.DebugAPIURL, &debugapi.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.DebugAPIInsecureTLS},
		}}})
	}
	if opts.Retry > 0 {
		c.retry = opts.Retry
	}

	return
}

// Addresses represents node's addresses
type Addresses struct {
	Overlay      swarm.Address
	Underlay     []string
	Ethereum     string
	PublicKey    string
	PSSPublicKey string
}

func (c *Client) Config() ClientOptions {
	return c.opts
}

// Addresses returns node's addresses
func (c *Client) Addresses(ctx context.Context) (resp Addresses, err error) {
	a, err := c.debug.Node.Addresses(ctx)
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

// Balance represents node's balance with peer
type Balance struct {
	Balance int64
	Peer    string
}

// Balance returns node's balance with a given peer
func (c *Client) Balance(ctx context.Context, a swarm.Address) (resp Balance, err error) {
	b, err := c.debug.Node.Balance(ctx, a)
	if err != nil {
		return Balance{}, fmt.Errorf("get balance with node %s: %w", a.String(), err)
	}

	return Balance{
		Balance: b.Balance,
		Peer:    b.Peer,
	}, nil
}

// Balances represents Balances's response
type Balances struct {
	Balances []Balance
}

// Balances returns node's balances
func (c *Client) Balances(ctx context.Context) (resp Balances, err error) {
	r, err := c.debug.Node.Balances(ctx)
	if err != nil {
		return Balances{}, fmt.Errorf("get balances: %w", err)
	}

	for _, b := range r.Balances {
		resp.Balances = append(resp.Balances, Balance{
			Balance: b.Balance,
			Peer:    b.Peer,
		})
	}

	return
}

// DownloadBytes downloads chunk from the node
func (c *Client) DownloadBytes(ctx context.Context, a swarm.Address) (data []byte, err error) {
	r, err := c.api.Bytes.Download(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadChunk downloads chunk from the node
func (c *Client) DownloadChunk(ctx context.Context, a swarm.Address, targets string) (data []byte, err error) {
	r, err := c.api.Chunks.Download(ctx, a, targets)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadFile downloads chunk from the node and returns it's size and hash
func (c *Client) DownloadFile(ctx context.Context, a swarm.Address) (size int64, hash []byte, err error) {
	r, err := c.api.Files.Download(ctx, a)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}

	h := fileHasher()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s, hashing copy: %w", a, err)
	}

	return size, h.Sum(nil), nil
}

// HasChunk returns true/false if node has a chunk
func (c *Client) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return c.debug.Node.HasChunk(ctx, a)
}

// Overlay returns node's overlay address
func (c *Client) Overlay(ctx context.Context) (o swarm.Address, err error) {
	var a debugapi.Addresses
	for r := 0; r < c.retry; r++ {
		time.Sleep(2 * time.Duration(r) * time.Second)

		a, err = c.debug.Node.Addresses(ctx)
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return swarm.Address{}, fmt.Errorf("get addresses: %w", err)
	}
	o = a.Overlay

	return
}

// Peers returns addresses of node's peers
func (c *Client) Peers(ctx context.Context) (peers []swarm.Address, err error) {
	ps, err := c.debug.Node.Peers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	for _, p := range ps.Peers {
		peers = append(peers, p.Address)
	}

	return
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
	r, err := c.debug.PingPong.Ping(ctx, node)
	if err != nil {
		return "", fmt.Errorf("ping node %s: %w", node, err)
	}
	return r.RTT, nil
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

// RemoveChunk removes chunk from the node
func (c *Client) RemoveChunk(ctx context.Context, a swarm.Address) error {
	return c.debug.Node.RemoveChunk(ctx, a)
}

// Settlement represents node's settlement with peer
type Settlement struct {
	Peer     string
	Received int
	Sent     int
}

// Settlement returns node's settlement with a given peer
func (c *Client) Settlement(ctx context.Context, a swarm.Address) (resp Settlement, err error) {
	b, err := c.debug.Node.Settlement(ctx, a)
	if err != nil {
		return Settlement{}, fmt.Errorf("get settlement with node %s: %w", a.String(), err)
	}

	return Settlement{
		Peer:     b.Peer,
		Received: b.Received,
		Sent:     b.Sent,
	}, nil
}

// CreatePostageBatch returns the batchID of a batch of postage stamps
func (c *Client) CreatePostageBatch(ctx context.Context, amount int64, depth uint64, label string) (string, error) {
	if depth < MinimumBatchDepth {
		depth = MinimumBatchDepth
	}
	return c.api.Postage.CreatePostageBatch(ctx, amount, depth, label)
}

func (c *Client) GetOrCreateBatch(ctx context.Context, depth uint64, sleep time.Duration) (string, error) {
	batches, err := c.PostageBatches(ctx)
	if err != nil {
		return "", err
	}

	fmt.Println("got batches", batches)

	if len(batches) != 0 {
		return batches[0].BatchID, nil
	}

	b, err := c.CreatePostageBatch(ctx, 1000, depth, "test-label") // todo: make amount optional, was hardcoded to 1
	time.Sleep(sleep)
	return b, err
}

// PostageBatches returns the list of batches of node
func (c *Client) PostageBatches(ctx context.Context) ([]api.PostageStampResponse, error) {
	return c.api.Postage.PostageBatches(ctx)
}

// ReserveState returns reserve radius, available capacity, inner and outer radiuses
func (c *Client) ReserveState(ctx context.Context) (debugapi.ReserveState, error) {
	return c.debug.Postage.Reservestate(ctx)
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
	TotalReceived int
	TotalSent     int
}

// Settlements returns node's settlements
func (c *Client) Settlements(ctx context.Context) (resp Settlements, err error) {
	r, err := c.debug.Node.Settlements(ctx)
	if err != nil {
		return Settlements{}, fmt.Errorf("get settlements: %w", err)
	}

	for _, b := range r.Settlements {
		resp.Settlements = append(resp.Settlements, Settlement{
			Peer:     b.Peer,
			Received: b.Received,
			Sent:     b.Sent,
		})
	}
	resp.TotalReceived = r.TotalReceived
	resp.TotalSent = r.TotalSent

	return
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
	r, err := c.debug.Node.CashoutStatus(ctx, a)
	if err != nil {
		return CashoutStatusResponse{}, fmt.Errorf("cashout: %w", err)
	}

	var cashoutStatusResult *CashoutStatusResult
	if r.Result != nil {
		cashoutStatusResult = &CashoutStatusResult{
			Recipient:  r.Result.Recipient,
			LastPayout: r.Result.LastPayout,
			Bounced:    r.Result.Bounced,
		}
	}

	return CashoutStatusResponse{
		Peer: r.Peer,
		Cheque: &Cheque{
			Beneficiary: r.Cheque.Beneficiary,
			Chequebook:  r.Cheque.Chequebook,
			Payout:      r.Cheque.Payout,
		},
		TransactionHash: r.TransactionHash,
		Result:          cashoutStatusResult,
		UncashedAmount:  r.UncashedAmount,
	}, nil
}

func (c *Client) Cashout(ctx context.Context, a swarm.Address) (resp string, err error) {
	r, err := c.debug.Node.Cashout(ctx, a)
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
	r, err := c.debug.Node.ChequebookBalance(ctx)
	if err != nil {
		return ChequebookBalanceResponse{}, fmt.Errorf("cashout: %w", err)
	}

	return ChequebookBalanceResponse{
		TotalBalance:     r.TotalBalance,
		AvailableBalance: r.AvailableBalance,
	}, nil
}

// Topology represents Kademlia topology
type Topology struct {
	Overlay        swarm.Address
	Connected      int
	Population     int
	NnLowWatermark int
	Depth          int
	Bins           map[string]Bin
	LightNodes     Bin
}

// Bin represents Kademlia bin
type Bin struct {
	Connected         int
	ConnectedPeers    []swarm.Address
	DisconnectedPeers []swarm.Address
	Population        int
}

// Topology returns Kademlia topology
func (c *Client) Topology(ctx context.Context) (topology Topology, err error) {
	var t debugapi.Topology
	for r := 0; r < c.retry; r++ {
		time.Sleep(2 * time.Duration(r) * time.Second)

		t, err = c.debug.Node.Topology(ctx)
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return Topology{}, fmt.Errorf("get topology: %w", err)
	}

	topology = Topology{
		Overlay:        t.BaseAddr,
		Connected:      t.Connected,
		Population:     t.Population,
		NnLowWatermark: t.NnLowWatermark,
		Depth:          t.Depth,
		Bins:           make(map[string]Bin),
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

	return
}

// Underlay returns node's underlay addresses
func (c *Client) Underlay(ctx context.Context) ([]string, error) {
	a, err := c.debug.Node.Addresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("get underlay: %w", err)
	}

	return a.Underlay, nil
}

// UploadBytes uploads bytes to the node
func (c *Client) WaitSync(ctx context.Context, UId uint32) error {
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
	h := fileHasher()
	r, err := c.api.Files.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), f.Size(), o)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// UploadCollection uploads TAR collection bytes to the node
func (c *Client) UploadCollection(ctx context.Context, f *File, o api.UploadOptions) (err error) {
	h := fileHasher()
	r, err := c.api.Dirs.Upload(ctx, io.TeeReader(f.DataReader(), h), f.Size(), o)
	if err != nil {
		return fmt.Errorf("upload collection: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}

// DownloadManifestFile downloads manifest file from the node and returns it's size and hash
func (c *Client) DownloadManifestFile(ctx context.Context, a swarm.Address, path string) (size int64, hash []byte, err error) {
	r, err := c.api.Dirs.Download(ctx, a, path)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file %s: %w", path, err)
	}

	h := fileHasher()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download manifest file %s: %w", path, err)
	}

	return size, h.Sum(nil), nil
}

// CreateTag creates tag on the node
func (c *Client) CreateTag(ctx context.Context) (resp api.TagResponse, err error) {
	resp, err = c.api.Tags.CreateTag(ctx)
	if err != nil {
		return resp, fmt.Errorf("create tag: %w", err)
	}

	return
}

// GetTag retrieves tag from node
func (c *Client) GetTag(ctx context.Context, tagUID uint32) (resp api.TagResponse, err error) {
	resp, err = c.api.Tags.GetTag(ctx, tagUID)
	if err != nil {
		return resp, fmt.Errorf("get tag: %w", err)
	}

	return
}
