package pss

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/push"
)

// Options represents pushsync check options
type Options struct {
	NodeGroup string
	NodeCount int
	Seed      int64
}

var (
	errDataMismatch        = errors.New("pss: data sent and received are not equal")
	errWebsocketConnection = errors.New("pss: websocket connection terminated with an error")
)

// Check sends a PSS message to random nodes while receiving them on the other end
func Check(c *bee.Cluster, o Options, pusher *push.Pusher, pushMetrics bool) (err error) {

	testData := []byte("Hello Swarm :)")
	testTopic := "test"
	testCount := o.NodeCount / 2

	pusher.Collector(sendAndReceiveGauge)

	ng := c.NodeGroup(o.NodeGroup)
	sortedNodes := ng.NodesSorted()

	set := randomDoubleSet(o.Seed, testCount, o.NodeCount)

	for i := 0; i < len(set); i++ {

		fmt.Printf("pss: test %d of %d\n", i+1, testCount)

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

		nodeAName := sortedNodes[set[i][0]]
		nodeBName := sortedNodes[set[i][1]]

		nodeA := ng.NodeClient(nodeAName)
		nodeB := ng.NodeClient(nodeBName)

		addrB, err := nodeB.Addresses(ctx)
		if err != nil {
			cancel()
			return err
		}

		ch, close, err := listenWebsocket(ctx, nodeB.Config().APIURL.Host, testTopic)
		if err != nil {
			cancel()
			return err
		}

		fmt.Printf("pss: sending test data to node %s and listening on node %s\n", nodeAName, nodeBName)

		tStart := time.Now()
		err = nodeA.SendPSSMessage(ctx, addrB.Overlay, addrB.PSSPublicKey, testTopic, testData)
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

		if pushMetrics {
			if err := pusher.Push(); err != nil {
				fmt.Printf("pss: push gauge: %v\n", err)
			}
		}
	}

	return nil
}

func randomDoubleSet(seed int64, count int, max int) [][]int {

	if max <= 1 {
		log.Fatal("max must be greater than one")
	}

	rnd := random.PseudoGenerator(seed)

	ret := make([][]int, 0)

	for i := 0; i < count; i++ {
		first := rnd.Intn(max)
		second := first

		for first == second {
			second = rnd.Intn(max)
		}

		ret = append(ret, []int{first, second})
	}

	return ret
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
