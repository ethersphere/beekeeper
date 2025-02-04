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
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/gorilla/websocket"
)

// Options represents check options
type Options struct {
	Count          int64
	AddressPrefix  int
	GasPrice       string
	PostageAmount  int64
	PostageTTL     time.Duration
	PostageDepth   uint64
	PostageLabel   string
	RequestTimeout time.Duration
	Seed           int64
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Count:          1,
		AddressPrefix:  1,
		GasPrice:       "",
		PostageAmount:  1,
		PostageTTL:     24 * time.Hour,
		PostageDepth:   16,
		PostageLabel:   "test-label",
		RequestTimeout: 5 * time.Minute,
		Seed:           random.Int64(),
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct {
	metrics metrics
	logger  logging.Logger
}

// NewCheck returns new check
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{
		metrics: newMetrics(),
		logger:  logger,
	}
}

func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	clients, err := cluster.NodesClients(ctx)
	if err != nil {
		return err
	}

	r := random.PseudoGenerator(o.Seed)
	fullNodeNames := cluster.FullNodeNames()

	if len(fullNodeNames) < 2 {
		return fmt.Errorf("pss test require at least 2 full nodes")
	}

	for i := 0; i < int(o.Count); i++ {
		c.logger.Infof("pss: test %d of %d", i+1, o.Count)

		nodeNameA := pickAtRandom(r, fullNodeNames, "")
		nodeNameB := pickAtRandom(r, fullNodeNames, nodeNameA)

		if err := c.testPss(nodeNameA, nodeNameB, clients, o); err != nil {
			return err
		}
	}

	return nil
}

func pickAtRandom(r *rand.Rand, names []string, skip string) string {
	for {
		i := r.Int31n(int32(len(names)))
		if names[i] != skip {
			return names[i]
		}
	}
}

var (
	errDataMismatch        = errors.New("pss: data sent and received are not equal")
	errWebsocketConnection = errors.New("pss: websocket connection terminated with an error")
)

var (
	testData  = []byte("Hello Swarm :)")
	testTopic = "test"
)

func (c *Check) testPss(nodeAName, nodeBName string, clients map[string]*bee.Client, o Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), o.RequestTimeout)

	nodeA := clients[nodeAName]
	nodeB := clients[nodeBName]

	addrB, err := nodeB.Addresses(ctx)
	if err != nil {
		cancel()
		return err
	}

	batchID, err := nodeA.GetOrCreateMutableBatch(ctx, o.PostageAmount, o.PostageDepth, o.PostageLabel)
	if err != nil {
		cancel()
		return fmt.Errorf("node %s: batched id %w", nodeAName, err)
	}
	c.logger.Infof("node %s: batched id %s", nodeAName, batchID)

	ch, close, err := listenWebsocket(ctx, nodeB.Host(), testTopic, c.logger)
	if err != nil {
		cancel()
		return err
	}

	c.logger.Infof("pss: sending test data to node %s and listening on node %s", nodeAName, nodeBName)

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
			c.logger.Info("pss: websocket connection received correct message")
			c.metrics.SendAndReceiveGauge.WithLabelValues(nodeAName, nodeBName).Set(time.Since(tStart).Seconds())
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

	return nil
}

func listenWebsocket(ctx context.Context, host string, topic string, logger logging.Logger) (<-chan string, func(), error) {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	ws, _, err := dialer.DialContext(ctx, fmt.Sprintf("ws://%s/pss/subscribe/%s", host, topic), http.Header{})
	if err != nil {
		return nil, nil, err
	}

	ch := make(chan string)

	go func() {
		_, data, err := ws.ReadMessage()
		if err != nil {
			logger.Infof("pss: websocket error %v", err)
			close(ch)
			return
		}

		ch <- string(data)
	}()

	return ch, func() { ws.Close() }, nil
}
