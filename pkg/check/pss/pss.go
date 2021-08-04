package pss

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents check options
type Options struct {
	AddressPrefix  int
	GasPrice       string
	MetricsPusher  *push.Pusher
	NodeCount      int
	PostageAmount  int64
	PostageDepth   uint64
	PostageLabel   string
	PostageWait    time.Duration
	RequestTimeout time.Duration
	Seed           int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		AddressPrefix:  1,
		GasPrice:       "",
		MetricsPusher:  nil,
		NodeCount:      1,
		PostageAmount:  1,
		PostageDepth:   16,
		PostageLabel:   "test-label",
		PostageWait:    5 * time.Second,
		RequestTimeout: 5 * time.Minute,
		Seed:           random.Int64(),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if o.MetricsPusher != nil {
		o.MetricsPusher.Collector(sendAndReceiveGauge)
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	sortedNodes := cluster.NodeNames()
	if o.NodeCount > len(sortedNodes) {
		o.NodeCount = len(sortedNodes)
	}

	for j, nodeAName := range shuffle(cluster.NodeNames()) {
		if j >= o.NodeCount {
			break
		}
		for _, nodeBName := range shuffle(cluster.FullNodeNames()) {
			if nodeAName == nodeBName {
				continue
			}

			fmt.Printf("pss: test %d of %d\n", j+1, o.NodeCount)

			if err := testPss(nodeAName, nodeBName, clients, o); err != nil {
				return err
			}

			break
		}
	}

	return nil
}

func shuffle(names []string) []string {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})
	return names
}

var (
	errDataMismatch        = errors.New("pss: data sent and received are not equal")
	errWebsocketConnection = errors.New("pss: websocket connection terminated with an error")
)

var (
	testData  = []byte("Hello Swarm :)")
	testTopic = "test"
)

func testPss(nodeAName, nodeBName string, clients map[string]*bee.Client, o Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), o.RequestTimeout)

	nodeA := clients[nodeAName]
	nodeB := clients[nodeBName]

	addrB, err := nodeB.Addresses(ctx)
	if err != nil {
		cancel()
		return err
	}

	batchID, err := nodeA.GetOrCreateBatch(ctx, o.PostageAmount, o.PostageDepth, o.GasPrice, o.PostageLabel)
	if err != nil {
		cancel()
		return fmt.Errorf("node %s: batched id %w", nodeAName, err)
	}
	fmt.Printf("node %s: batched id %s\n", nodeAName, batchID)
	time.Sleep(o.PostageWait)

	ch, close, err := listenWebsocket(ctx, nodeB.Config().APIURL.Host, testTopic)
	if err != nil {
		cancel()
		return err
	}

	fmt.Printf("pss: sending test data to node %s and listening on node %s\n", nodeAName, nodeBName)

	tStart := time.Now()
	err = nodeA.SendPSSMessage(ctx, addrB.Overlay, addrB.PSSPublicKey, testTopic, o.AddressPrefix, testData, batchID)
	if err != nil {
		close()
		cancel()
		return err
	}

	msg, ok := <-ch
	if ok {
		if msg == string(testData) {
			fmt.Println("pss: websocket connection received correct message")
			sendAndReceiveGauge.WithLabelValues(nodeAName, nodeBName).Set(time.Since(tStart).Seconds())
		} else {
			err = errDataMismatch
		}
	} else {
		err = errWebsocketConnection
	}

	cancel()
	close()

	if err != nil {
		return err
	}

	if o.MetricsPusher != nil {
		if err := o.MetricsPusher.Push(); err != nil {
			fmt.Printf("pss: push gauge: %v\n", err)
		}
	}

	return nil
}

func listenWebsocket(ctx context.Context, host string, topic string) (<-chan string, func(), error) {

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	ws, _, err := dialer.DialContext(ctx, fmt.Sprintf("ws://%s/pss/subscribe/%s", host, topic), nil)
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan string)

	go func() {
		_, data, err := ws.ReadMessage()
		if err != nil {
			fmt.Printf("pss: websocket error %v\n", err)
			close(ch)
			return
		}

		ch <- string(data)
	}()

	return ch, func() { ws.Close() }, nil
}
