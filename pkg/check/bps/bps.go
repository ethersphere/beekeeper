// Package bps checks the BPS live stream end to end through the Bee API: a
// publisher signs updates on one node, a broker fans them out, and subscribers
// on other nodes receive and verify them. It covers what needs real nodes and
// real p2p streams; the broker's rules are covered by Bee's unit tests.
package bps

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/ethersphere/beekeeper/pkg/random"
)

// Options represents check options
type Options struct {
	Messages    int           // updates published before and after the publisher reconnects
	Seed        int64         // seed of the node selection
	StepTimeout time.Duration // bound on every single wait
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		Messages:    5,
		StepTimeout: 30 * time.Second,
	}
}

// maxMessages is how many deliveries a socket buffers: the publisher sends a
// whole batch before the check reads the subscribers.
const maxMessages = 64

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance.
type Check struct {
	logger logging.Logger
}

// NewCheck returns a new check instance.
func NewCheck(logger logging.Logger) beekeeper.Action {
	return &Check{logger: logger}
}

// Run publishes a live stream through a broker of the cluster and requires
// that:
//
//  1. subscribers on other nodes receive every update, in order, before and
//     after the publisher reconnects;
//  2. a node that is not a broker refuses the join;
//  3. once every socket is closed, the nodes hold no more bps goroutines and
//     the broker no more cohorts than before;
//  4. the nodes are still connected to the broker and no side has blocklisted
//     the other.
//
// Every step runs even when an earlier one fails, so one run reports them all.
func (c *Check) Run(ctx context.Context, cluster orchestration.Cluster, opts any) error {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}
	if o.Messages <= 0 || o.Messages > maxMessages {
		return fmt.Errorf("messages must be between 1 and %d", maxMessages)
	}

	c.logger.Infof("bps: seed %d", o.Seed)
	r, err := pickRoles(ctx, cluster, random.PseudoGenerator(o.Seed))
	if err != nil {
		return err
	}
	c.logger.Infof("bps: broker %s, publisher %s, subscribers %s, not a broker %s",
		r.broker.Name(), r.publisher.Name(), names(r.subscribers), r.notBroker.Name())

	before, err := r.usage(ctx)
	if err != nil {
		return err
	}

	var errs []error
	if err := c.liveStream(ctx, r, o); err != nil {
		errs = append(errs, fmt.Errorf("live stream: %w", err))
	}
	if err := c.notABroker(ctx, r); err != nil {
		errs = append(errs, fmt.Errorf("not a broker: %w", err))
	}
	if err := c.released(ctx, r, before, o.StepTimeout); err != nil {
		errs = append(errs, fmt.Errorf("release: %w", err))
	}
	if err := c.connected(ctx, r); err != nil {
		errs = append(errs, fmt.Errorf("connectivity: %w", err))
	}
	return errors.Join(errs...)
}

// roles are the nodes a run uses.
type roles struct {
	broker        *bee.Client
	brokerAddr    swarm.Address
	publisher     *bee.Client
	subscribers   []*bee.Client
	notBroker     *bee.Client
	notBrokerAddr swarm.Address
	overlays      map[string]swarm.Address
}

// clients are the nodes that hold a socket during the run.
func (r roles) clients() []*bee.Client {
	return append([]*bee.Client{r.publisher}, r.subscribers...)
}

// nodes are the broker and the clients: every node that serves the run.
func (r roles) nodes() []*bee.Client {
	return append([]*bee.Client{r.broker}, r.clients()...)
}

func names(clients []*bee.Client) string {
	l := make([]string, 0, len(clients))
	for _, cl := range clients {
		l = append(l, cl.Name())
	}
	return strings.Join(l, ", ")
}

// pickRoles picks a broker, and the publisher and subscribers among the nodes
// already connected to it: a node opens a broker stream only to a peer it is
// connected to, and the check does not connect nodes itself. A light node is
// preferred as one of the subscribers. The node that is not a broker is a peer
// of the publisher, so that its refusal is the protocol's and not a missing
// connection.
func pickRoles(ctx context.Context, cluster orchestration.Cluster, rnd *rand.Rand) (roles, error) {
	nodes := cluster.Nodes()
	all := make([]string, 0, len(nodes))
	for name := range nodes {
		all = append(all, name)
	}
	slices.Sort(all)
	rnd.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	overlays := make(map[string]swarm.Address, len(all))
	peers := make(map[string]map[string]struct{}, len(all))
	var brokers []string
	for _, name := range all {
		cl := nodes[name].Client()
		overlay, err := cl.Overlay(ctx)
		if err != nil {
			return roles{}, fmt.Errorf("node %s: %w", name, err)
		}
		overlays[name] = overlay
		ps, err := cl.Peers(ctx)
		if err != nil {
			return roles{}, fmt.Errorf("node %s: %w", name, err)
		}
		peers[name] = make(map[string]struct{}, len(ps))
		for _, p := range ps {
			peers[name][p.ByteString()] = struct{}{}
		}
		if isBroker(nodes[name]) {
			brokers = append(brokers, name)
		}
	}
	if len(brokers) == 0 {
		return roles{}, errors.New("bps: no node of the cluster has bps-broker set")
	}
	connectedTo := func(name, target string) bool {
		_, ok := peers[name][overlays[target].ByteString()]
		return ok
	}

	// the broker with the most connected nodes that can hold a socket first
	connected := func(broker string) (n int) {
		for _, name := range all {
			if cfg := nodes[name].Config(); name != broker && cfg != nil && !cfg.IsBootnode() && connectedTo(name, broker) {
				n++
			}
		}
		return n
	}
	slices.SortStableFunc(brokers, func(a, b string) int { return connected(b) - connected(a) })

	for _, broker := range brokers {
		var full, light []string
		for _, name := range all {
			cfg := nodes[name].Config()
			if name == broker || cfg == nil || cfg.IsBootnode() || !connectedTo(name, broker) {
				continue
			}
			if cfg.IsFullNode() {
				full = append(full, name)
			} else {
				light = append(light, name)
			}
		}
		if len(full) == 0 || len(full)+len(light) < 2 {
			continue
		}
		publisher := full[0]
		subscribers := full[1:]
		if len(light) > 0 {
			subscribers = append([]string{light[0]}, subscribers...)
		}
		if len(subscribers) > 2 {
			subscribers = subscribers[:2]
		}
		for _, name := range all {
			if name == broker || name == publisher || isBroker(nodes[name]) || !connectedTo(publisher, name) {
				continue
			}
			r := roles{
				broker:        nodes[broker].Client(),
				brokerAddr:    overlays[broker],
				publisher:     nodes[publisher].Client(),
				notBroker:     nodes[name].Client(),
				notBrokerAddr: overlays[name],
				overlays:      overlays,
			}
			for _, s := range subscribers {
				r.subscribers = append(r.subscribers, nodes[s].Client())
			}
			return r, nil
		}
	}
	return roles{}, fmt.Errorf("bps: none of the brokers %v has two connected nodes, one of them a full node connected to a node that is not a broker", brokers)
}

func isBroker(n orchestration.Node) bool {
	cfg := n.Config()
	return cfg != nil && cfg.BpsBroker != nil && *cfg.BpsBroker
}

// liveStream publishes Messages updates, reconnects the publisher, publishes
// Messages more, and requires every subscriber to receive all of them in order.
func (c *Check) liveStream(ctx context.Context, r roles, o Options) error {
	ch, err := newChannel()
	if err != nil {
		return err
	}

	var subs []*socket
	defer func() {
		for _, s := range subs {
			s.close()
		}
	}()
	for _, node := range r.subscribers {
		owner, _, err := newOwner()
		if err != nil {
			return err
		}
		s, err := dial(ctx, node, ch, r.brokerAddr, owner)
		if err != nil {
			return fmt.Errorf("subscriber: %w", err)
		}
		subs = append(subs, s)
	}

	n := uint64(o.Messages)
	first, err := c.publish(ctx, r, o, ch, subs, 0, n)
	if err != nil {
		return err
	}
	c.logger.Infof("bps: %d updates delivered in order to %d subscribers", n, len(subs))

	second, err := c.publish(ctx, r, o, ch, subs, n, 2*n)
	if err != nil {
		return fmt.Errorf("after the publisher reconnected: %w", err)
	}
	if string(first) == string(second) {
		return errors.New("the reconnected publisher was given the challenge it had before")
	}
	c.logger.Infof("bps: publisher reconnected under a new challenge, %d more updates delivered", n)
	return nil
}

// publish opens the publisher's socket, authenticates, publishes the updates
// in [from, to) and waits until every subscriber has them. It returns the
// challenge the publisher was given.
func (c *Check) publish(ctx context.Context, r roles, o Options, ch *channel, subs []*socket, from, to uint64) ([]byte, error) {
	pub, err := dial(ctx, r.publisher, ch, r.brokerAddr, ch.principal)
	if err != nil {
		return nil, fmt.Errorf("publisher: %w", err)
	}
	defer pub.close()

	if err := pub.publish(ch, kindAuth, 0, nil); err != nil {
		return nil, err
	}
	for i := from; i < to; i++ {
		if err := pub.publish(ch, kindData, i, payloadOf(i)); err != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, o.StepTimeout)
	defer cancel()
	for _, s := range subs {
		for i := from; i < to; i++ {
			if err := s.expect(ctx, ch, pub.challenge, i, payloadOf(i)); err != nil {
				return nil, err
			}
		}
	}
	return pub.challenge, nil
}

func payloadOf(index uint64) []byte {
	return fmt.Appendf(nil, "update %d", index)
}

// notABroker requires that a join at a connected node that runs no broker
// closes the socket with the code of a lost broker stream.
func (c *Check) notABroker(ctx context.Context, r roles) error {
	ch, err := newChannel()
	if err != nil {
		return err
	}
	s, err := dial(ctx, r.publisher, ch, r.notBrokerAddr, ch.principal)
	if err == nil {
		s.close()
		return fmt.Errorf("node %s accepted the join", r.notBroker.Name())
	}
	var ce *closedError
	if !errors.As(err, &ce) {
		return err
	}
	if ce.code != closeBrokerGone {
		return fmt.Errorf("close code %d %q, want %d", ce.code, ce.text, closeBrokerGone)
	}
	c.logger.Infof("bps: join at %s refused: %d %q", r.notBroker.Name(), ce.code, ce.text)
	return nil
}

// usage is what the run's nodes hold for bps.
type usage struct {
	cohorts    float64        // the broker's bee_bps_cohorts
	goroutines map[string]int // bps goroutines per node, the broker included
}

func (r roles) usage(ctx context.Context) (usage, error) {
	u := usage{goroutines: make(map[string]int)}
	metrics, err := r.broker.API().Node.Metrics(ctx)
	if err != nil {
		return u, fmt.Errorf("node %s: metrics: %w", r.broker.Name(), err)
	}
	if u.cohorts, err = metric(metrics, "bee_bps_cohorts"); err != nil {
		return u, fmt.Errorf("node %s: %w", r.broker.Name(), err)
	}
	for _, cl := range r.nodes() {
		dump, err := cl.API().Node.Goroutines(ctx)
		if err != nil {
			return u, fmt.Errorf("node %s: goroutines: %w", cl.Name(), err)
		}
		u.goroutines[cl.Name()] = bpsGoroutines(dump)
	}
	return u, nil
}

// metric returns the value of the metric that has no labels.
func metric(metrics, name string) (float64, error) {
	for line := range strings.SplitSeq(metrics, "\n") {
		if v, ok := strings.CutPrefix(line, name+" "); ok {
			return strconv.ParseFloat(strings.TrimSpace(v), 64)
		}
	}
	return 0, fmt.Errorf("metric %s not found", name)
}

// bpsGoroutines counts the goroutines of a dump that a bps session can leave
// behind: those that run the protocol (pkg/bps) or the WebSocket bridge (the
// bps functions of pkg/api), and stream readers nobody waits for. A stream
// reader is started by ReadMsgWithContext; while its caller still waits in
// that call the read is live, as the long reads of pullsync are, and it is not
// counted.
func bpsGoroutines(dump string) (n int) {
	const (
		reader  = "protobuf.Reader.ReadMsgWithContext.func1("
		waiting = "protobuf.Reader.ReadMsgWithContext("
		creator = "ReadMsgWithContext in goroutine "
	)
	// the bridge's functions and methods are all named bps...
	own := []string{"/pkg/bps.", "/pkg/api.bps", "/pkg/api.(*Service).bps"}
	waiters := make(map[string]struct{})
	var readers []string // the creator of every stream reader
	for stack := range strings.SplitSeq(dump, "\n\n") {
		switch {
		case slices.ContainsFunc(own, func(f string) bool { return strings.Contains(stack, f) }):
			n++
		case strings.Contains(stack, reader):
			_, id, _ := strings.Cut(stack, creator)
			id, _, _ = strings.Cut(id, "\n")
			readers = append(readers, id)
		case strings.Contains(stack, waiting):
			id, _, _ := strings.Cut(strings.TrimPrefix(stack, "goroutine "), " ")
			waiters[id] = struct{}{}
		}
	}
	for _, id := range readers {
		if _, ok := waiters[id]; !ok {
			n++
		}
	}
	return n
}

// released waits until the nodes hold no more for bps than before the run.
func (c *Check) released(ctx context.Context, r roles, before usage, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var held []string
	for {
		now, err := r.usage(ctx)
		switch {
		case ctx.Err() != nil: // keep what the last full reading found
		case err != nil:
			held = []string{err.Error()}
		default:
			held = held[:0]
			if now.cohorts > before.cohorts {
				held = append(held, fmt.Sprintf("broker %s holds %g cohorts, %g before the run", r.broker.Name(), now.cohorts, before.cohorts))
			}
			for _, cl := range r.nodes() {
				if d := now.goroutines[cl.Name()] - before.goroutines[cl.Name()]; d > 0 {
					held = append(held, fmt.Sprintf("node %s has %d more bps goroutines than before the run", cl.Name(), d))
				}
			}
			if len(held) == 0 {
				c.logger.Infof("bps: every socket closed, nothing is held: %g cohorts, goroutines %v", now.cohorts, now.goroutines)
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s after every socket was closed: %s", timeout, strings.Join(held, "; "))
		case <-ticker.C:
		}
	}
}

// connected requires that the run left every client connected to the broker
// and that neither has blocklisted the other.
func (c *Check) connected(ctx context.Context, r roles) error {
	var errs []error
	brokerList, err := r.broker.API().Node.Blocklist(ctx)
	if err != nil {
		return fmt.Errorf("node %s: blocklist: %w", r.broker.Name(), err)
	}
	for _, cl := range r.clients() {
		peers, err := cl.Peers(ctx)
		if err != nil {
			return fmt.Errorf("node %s: %w", cl.Name(), err)
		}
		if !slices.ContainsFunc(peers, r.brokerAddr.Equal) {
			errs = append(errs, fmt.Errorf("node %s is no longer connected to broker %s", cl.Name(), r.broker.Name()))
		}
		list, err := cl.API().Node.Blocklist(ctx)
		if err != nil {
			return fmt.Errorf("node %s: blocklist: %w", cl.Name(), err)
		}
		for _, p := range list.Peers {
			if p.Address.Equal(r.brokerAddr) {
				errs = append(errs, fmt.Errorf("node %s blocklisted broker %s for %d s: %s", cl.Name(), r.broker.Name(), p.Duration, p.Reason))
			}
		}
		for _, p := range brokerList.Peers {
			if p.Address.Equal(r.overlays[cl.Name()]) {
				errs = append(errs, fmt.Errorf("broker %s blocklisted node %s for %d s: %s", r.broker.Name(), cl.Name(), p.Duration, p.Reason))
			}
		}
	}
	if len(errs) == 0 {
		c.logger.Infof("bps: %s still connected to broker %s, nothing blocklisted", names(r.clients()), r.broker.Name())
	}
	return errors.Join(errs...)
}
