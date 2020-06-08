package pingpong

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	pingCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			// Namespace: "svetomir",
			// Subsystem: "pingpong",
			Name: "ping_count_total",
			Help: "Total ping count",
		},
		[]string{"node_index", "peer_index", "node_address", "peer_address"},
	)
	rttGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			// Namespace: "svetomir",
			// Subsystem: "pingpong",
			Name: "ping_rtt_gauge_seconds",
			Help: "Round-trip time of a ping",
		},
		[]string{"node_index", "peer_index", "node_address", "peer_address"},
	)
	rttHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			// Namespace: "svetomir",
			// Subsystem: "pingpong",
			Name:    "ping_rtt_histogram_seconds",
			Help:    "Round-trip time of a ping",
			Buckets: prometheus.LinearBuckets(2, 2, 10),
		},
		[]string{"node_index", "peer_index", "node_address", "peer_address"},
	)
)

// Check executes ping from all nodes to all other nodes in the cluster
func Check(cluster bee.Cluster, pusher *push.Pusher) (err error) {
	ctx := context.Background()

	pusher.Collector(pingCount)
	pusher.Collector(rttGauge)
	pusher.Collector(rttHistogram)

	for n := range nodeStream(ctx, cluster.Nodes) {
		if n.Error != nil {
			fmt.Printf("node %d: %s\n", n.Index, n.Error)
			continue
		}
		fmt.Printf("Node %d. Peer %d RTT: %s. Node: %s Peer: %s\n", n.Index, n.PeerIndex, n.RTT, n.Address, n.PeerAddress)

		pingCount.WithLabelValues(strconv.Itoa(n.Index), strconv.Itoa(n.PeerIndex), n.Address.String(), n.PeerAddress.String()).Inc()
		rtt, err := rttParse(n.RTT)
		if err != nil {
			fmt.Printf("node %d: %s\n", n.Index, err)
			continue
		}
		rttGauge.WithLabelValues(strconv.Itoa(n.Index), strconv.Itoa(n.PeerIndex), n.Address.String(), n.PeerAddress.String()).Set(rtt)
		rttHistogram.WithLabelValues(strconv.Itoa(n.Index), strconv.Itoa(n.PeerIndex), n.Address.String(), n.PeerAddress.String()).Observe(rtt)

		if err := pusher.Push(); err != nil {
			fmt.Printf("node %d: %s\n", n.Index, err)
		}
	}

	return
}

type nodeStreamMsg struct {
	Index       int
	Address     swarm.Address
	PeerIndex   int
	PeerAddress swarm.Address
	RTT         string
	Error       error
}

func nodeStream(ctx context.Context, nodes []bee.Node) <-chan nodeStreamMsg {
	nodeStream := make(chan nodeStreamMsg)

	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(i int, node bee.Node) {
			defer wg.Done()

			address, err := node.Overlay(ctx)
			if err != nil {
				nodeStream <- nodeStreamMsg{Index: i, Error: err}
				return
			}

			peers, err := node.Peers(ctx)
			if err != nil {
				nodeStream <- nodeStreamMsg{Index: i, Error: err}
				return
			}

			for m := range node.PingStream(ctx, peers) {
				if m.Error != nil {
					nodeStream <- nodeStreamMsg{Index: i, Error: m.Error}
				}
				nodeStream <- nodeStreamMsg{
					Index:       i,
					Address:     address,
					PeerIndex:   m.Index,
					PeerAddress: m.Node,
					RTT:         m.RTT,
				}
			}
		}(i, node)
	}

	go func() {
		wg.Wait()
		close(nodeStream)
	}()

	return nodeStream
}

func rttParse(rtt string) (float64, error) {
	rttMs, err := strconv.ParseFloat(strings.Split(rtt, "ms")[0], 64)
	if err != nil {
		return 0, err
	}
	return rttMs / 1000, err
}
