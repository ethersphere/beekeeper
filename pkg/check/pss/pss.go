package pss

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/ethersphere/beekeeper/pkg/wslistener"
)

// Options represents check options
type Options struct {
	Count          int64
	AddressPrefix  int
	GasPrice       string
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
	testData  = []byte("Hello Swarm :)")
	testTopic = "test"
)

func (c *Check) testPss(nodeAName, nodeBName string, clients map[string]*bee.Client, o Options) error {
	ctx, cancel := context.WithTimeout(context.Background(), o.RequestTimeout)
	defer cancel()

	nodeA := clients[nodeAName]
	nodeB := clients[nodeBName]

	addrB, err := nodeB.Addresses(ctx)
	if err != nil {
		return err
	}

	batchID, err := nodeA.GetOrCreateMutableBatch(ctx, o.PostageTTL, o.PostageDepth, o.PostageLabel)
	if err != nil {
		return fmt.Errorf("node %s: batched id %w", nodeAName, err)
	}
	c.logger.Infof("node %s: batched id %s", nodeAName, batchID)

	ch, closer, err := wslistener.ListenWebSocket(ctx, nodeB, fmt.Sprintf("/pss/subscribe/%s", testTopic), c.logger)
	if err != nil {
		return err
	}

	c.logger.Infof("pss: sending test data to node %s and listening on node %s", nodeAName, nodeBName)

	defer closer()

	tStart := time.Now()
	err = nodeA.SendPSSMessage(ctx, addrB.Overlay, addrB.PSSPublicKey, testTopic, o.AddressPrefix, testData, batchID)
	if err != nil {
		return err
	}
	c.logger.Infof("pss: test data sent successfully to node %s. Waiting for response from node %s", nodeAName, nodeBName)

	for {
		select {
		case <-time.After(1 * time.Minute):
			return fmt.Errorf("correct message not received after %s", 1*time.Minute)
		default:
			msg, ok := <-ch
			if !ok {
				return fmt.Errorf("ws closed before receiving correct message")
			}

			if msg == string(testData) {
				c.logger.Info("pss: websocket connection received correct message")
				c.metrics.SendAndReceiveGauge.WithLabelValues(nodeAName, nodeBName).Set(time.Since(tStart).Seconds())
				return nil
			}
			c.logger.Infof("pss: received incorrect message. trying again. want %s, got %s", string(testData), msg)
		}
	}
}
