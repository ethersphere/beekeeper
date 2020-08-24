package bee

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
	bmtlegacy "github.com/ethersphere/bmt/legacy"
)

// Node represents Bee node
type Node struct {
	api   *api.Client
	debug *debugapi.Client
}

// NodeOptions represents Bee node options
type NodeOptions struct {
	APIURL              *url.URL
	APIInsecureTLS      bool
	DebugAPIURL         *url.URL
	DebugAPIInsecureTLS bool
}

// NewNode returns new node
func NewNode(opts NodeOptions) Node {
	return Node{
		api: api.NewClient(opts.APIURL, &api.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.APIInsecureTLS},
		}}}),
		debug: debugapi.NewClient(opts.DebugAPIURL, &debugapi.ClientOptions{HTTPClient: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.DebugAPIInsecureTLS},
		}}}),
	}
}

// Addresses represents node's addresses
type Addresses struct {
	Overlay  swarm.Address
	Underlay []string
}

// Addresses returns node's addresses
func (n *Node) Addresses(ctx context.Context) (resp Addresses, err error) {
	a, err := n.debug.Node.Addresses(ctx)
	if err != nil {
		return Addresses{}, fmt.Errorf("get addresses: %w", err)
	}

	return Addresses{
		Overlay:  a.Overlay,
		Underlay: a.Underlay,
	}, nil
}

// DownloadBytes downloads chunk from the node
func (n *Node) DownloadBytes(ctx context.Context, a swarm.Address) (data []byte, err error) {
	r, err := n.api.Bytes.Download(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadChunk downloads chunk from the node
func (n *Node) DownloadChunk(ctx context.Context, a swarm.Address, targets string) (data []byte, err error) {
	r, err := n.api.Chunks.Download(ctx, a, targets)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
}

// DownloadFile downloads chunk from the node and returns it's size and hash
func (n *Node) DownloadFile(ctx context.Context, a swarm.Address) (size int64, hash []byte, err error) {
	r, err := n.api.Files.Download(ctx, a)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}

	h := fileHahser()
	size, err = io.Copy(h, r)
	if err != nil {
		return 0, nil, fmt.Errorf("download file %s: %w", a, err)
	}

	return size, h.Sum(nil), nil
}

// HasChunk returns true/false if node has a chunk
func (n *Node) HasChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return n.debug.Node.HasChunk(ctx, a)
}

// Overlay returns node's overlay address
func (n *Node) Overlay(ctx context.Context) (swarm.Address, error) {
	a, err := n.debug.Node.Addresses(ctx)
	if err != nil {
		return swarm.Address{}, fmt.Errorf("get overlay: %w", err)
	}

	return a.Overlay, nil
}

// Peers returns addresses of node's peers
func (n *Node) Peers(ctx context.Context) (peers []swarm.Address, err error) {
	ps, err := n.debug.Node.Peers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	for _, p := range ps.Peers {
		peers = append(peers, p.Address)
	}

	return
}

// PinChunk returns true/false if chunk pinning is successful
func (n *Node) PinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return n.debug.Node.PinChunk(ctx, a)
}

// PinnedChunk represents pinned chunk
type PinnedChunk struct {
	Address    swarm.Address
	PinCounter int
}

// PinnedChunk returns pinned chunk
func (n *Node) PinnedChunk(ctx context.Context, a swarm.Address) (PinnedChunk, error) {
	p, err := n.debug.Node.PinnedChunk(ctx, a)
	if err != nil {
		return PinnedChunk{}, fmt.Errorf("get pinned chunk: %w", err)
	}

	return PinnedChunk{
		Address:    p.Address,
		PinCounter: p.PinCounter,
	}, nil
}

// PinnedChunks represents pinned chunks
type PinnedChunks struct {
	Chunks []PinnedChunk
}

// PinnedChunks returns pinned chunks
func (n *Node) PinnedChunks(ctx context.Context) (PinnedChunks, error) {
	p, err := n.debug.Node.PinnedChunks(ctx)
	if err != nil {
		return PinnedChunks{}, fmt.Errorf("get pinned chunks: %w", err)
	}

	r := PinnedChunks{}
	for _, c := range p.Chunks {
		r.Chunks = append(r.Chunks, PinnedChunk{
			Address:    c.Address,
			PinCounter: c.PinCounter,
		})
	}

	return r, nil
}

// Ping pings other node
func (n *Node) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := n.debug.PingPong.Ping(ctx, node)
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
func (n *Node) PingStream(ctx context.Context, nodes []swarm.Address) <-chan PingStreamMsg {
	pingStream := make(chan PingStreamMsg)

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node swarm.Address) {
			defer wg.Done()

			rtt, err := n.Ping(ctx, node)
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

// Topology represents Kademlia topology
type Topology struct {
	Overlay        swarm.Address
	Connected      int
	Population     int
	NnLowWatermark int
	Depth          int
	Bins           map[string]Bin
}

// Bin represents Kademlia bin
type Bin struct {
	Connected         int
	ConnectedPeers    []swarm.Address
	DisconnectedPeers []swarm.Address
	Population        int
}

// Topology returns Kademlia topology
func (n *Node) Topology(ctx context.Context) (topology Topology, err error) {
	t, err := n.debug.Node.Topology(ctx)
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

	return
}

// Underlay returns node's underlay addresses
func (n *Node) Underlay(ctx context.Context) ([]string, error) {
	a, err := n.debug.Node.Addresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("get underlay: %w", err)
	}

	return a.Underlay, nil
}

// UnpinChunk returns true/false if chunk unpinning is successful
func (n *Node) UnpinChunk(ctx context.Context, a swarm.Address) (bool, error) {
	return n.debug.Node.UnpinChunk(ctx, a)
}

// UploadBytes uploads chunk to the node
func (n *Node) UploadBytes(ctx context.Context, c *Chunk) (err error) {
	r, err := n.api.Bytes.Upload(ctx, bytes.NewReader(c.Data()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	c.address = r.Reference

	return
}

// UploadChunk uploads chunk to the node
func (n *Node) UploadChunk(ctx context.Context, c *Chunk) (err error) {
	p := bmtlegacy.NewTreePool(chunkHahser, swarm.Branches, bmtlegacy.PoolSize)
	hasher := bmtlegacy.New(p)
	err = hasher.SetSpan(int64(c.Span()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}
	_, err = hasher.Write(c.Data()[8:])
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}
	c.address = swarm.NewAddress(hasher.Sum(nil))

	_, err = n.api.Chunks.Upload(ctx, c.address, bytes.NewReader(c.Data()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	return
}

// RemoveChunk removes the given chunk from the node's local store
func (n *Node) RemoveChunk(ctx context.Context, c *Chunk) (err error) {
	return n.debug.Chunks.Remove(ctx, c.Address())
}

// UploadFile uploads file to the node
func (n *Node) UploadFile(ctx context.Context, f *File) (err error) {
	h := fileHahser()
	r, err := n.api.Files.Upload(ctx, f.Name(), io.TeeReader(f.DataReader(), h), f.Size())
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	f.address = r.Reference
	f.hash = h.Sum(nil)

	return
}
