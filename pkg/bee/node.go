package bee

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beeclient/debugapi"
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

// DownloadChunk downloads chunk from the node
func (n *Node) DownloadChunk(ctx context.Context, a swarm.Address) (data []byte, err error) {
	r, err := n.api.Bzz.Download(ctx, a)
	if err != nil {
		return nil, fmt.Errorf("download chunk %s: %w", a, err)
	}

	return ioutil.ReadAll(r)
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

// Ping pings other node
func (n *Node) Ping(ctx context.Context, node swarm.Address) (rtt string, err error) {
	r, err := n.api.PingPong.Ping(ctx, node)
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

// UploadChunk uploads chunk to the node
func (n *Node) UploadChunk(ctx context.Context, c *Chunk) (err error) {
	r, err := n.api.Bzz.Upload(ctx, bytes.NewReader(c.Data()))
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	c.address = r.Hash

	return
}
