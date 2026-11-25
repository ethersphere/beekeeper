package bps

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/gorilla/websocket"
)

// The chunk kinds, as one byte on the WebSocket and in the id's prefix.
const (
	kindData byte = 1
	kindAuth byte = 2
)

// closeBrokerGone is the close code of a socket whose broker stream ended.
const closeBrokerGone = 4003

const (
	challengeSize  = 32
	ownerSize      = 20
	deliveryHeader = 1 + ownerSize + challengeSize + 8 // kind | owner | challenge | index
	servicePrefix  = "bps-service:v1"
	dialTimeout    = 30 * time.Second
)

// channel is one cohort: a topic and the key of its principal.
type channel struct {
	topic     []byte
	signer    crypto.Signer
	principal []byte
}

func newChannel() (*channel, error) {
	topic := make([]byte, swarm.HashSize)
	if _, err := rand.Read(topic); err != nil {
		return nil, err
	}
	principal, signer, err := newOwner()
	if err != nil {
		return nil, err
	}
	return &channel{topic: topic, signer: signer, principal: principal}, nil
}

func newOwner() ([]byte, crypto.Signer, error) {
	key, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return nil, nil, err
	}
	signer := crypto.NewDefaultSigner(key)
	addr, err := signer.EthereumAddress()
	if err != nil {
		return nil, nil, err
	}
	return addr.Bytes(), signer, nil
}

// sessionTopic is keccak256(prefix | topic | challenge); the prefix is empty
// for DATA and "bps-service:v1" | kind for any other kind.
func sessionTopic(kind byte, topic, challenge []byte) ([]byte, error) {
	var b []byte
	if kind != kindData {
		b = append(b, servicePrefix...)
		b = append(b, kind)
	}
	b = append(b, topic...)
	b = append(b, challenge...)
	return crypto.LegacyKeccak256(b)
}

// chunkID is keccak256(sessionTopic | index), the index big-endian. The check
// derives it here instead of importing Bee's bps package, so a change of the
// derivation in Bee does not go unnoticed.
func chunkID(kind byte, topic, challenge []byte, index uint64) ([]byte, error) {
	st, err := sessionTopic(kind, topic, challenge)
	if err != nil {
		return nil, err
	}
	return crypto.LegacyKeccak256(binary.BigEndian.AppendUint64(st, index))
}

// delivery is one binary frame the node hands a subscriber.
type delivery struct {
	kind      byte
	owner     []byte
	challenge []byte
	index     uint64
	soc       []byte
}

// payload checks that the delivery is the principal's DATA chunk at the id
// derived from its challenge and index, and returns what the chunk carries.
func (d delivery) payload(ch *channel) ([]byte, error) {
	if d.kind != kindData {
		return nil, fmt.Errorf("delivery of kind %d", d.kind)
	}
	want, err := chunkID(d.kind, ch.topic, d.challenge, d.index)
	if err != nil {
		return nil, err
	}
	s, err := soc.FromChunk(swarm.NewChunk(swarm.ZeroAddress, d.soc))
	if err != nil {
		return nil, fmt.Errorf("parse soc: %w", err)
	}
	if !bytes.Equal(s.ID(), want) {
		return nil, fmt.Errorf("chunk id %x, derived %x", s.ID(), want)
	}
	if !bytes.Equal(s.OwnerAddress(), ch.principal) || !bytes.Equal(d.owner, ch.principal) {
		return nil, fmt.Errorf("chunk signed by %x and delivered as %x, want %x", s.OwnerAddress(), d.owner, ch.principal)
	}
	return s.WrappedChunk().Data()[swarm.SpanSize:], nil
}

// closedError is a socket the node closed.
type closedError struct {
	node string
	code int
	text string
}

func (c *closedError) Error() string {
	return fmt.Sprintf("node %s closed the socket: code %d %q", c.node, c.code, c.text)
}

// socket is one WebSocket to a node's bps endpoint.
type socket struct {
	node      string
	ws        *websocket.Conn
	writeMu   sync.Mutex
	challenge []byte

	challenges chan string
	deliveries chan delivery
	closed     chan struct{} // closed when the read loop ends
	closeCode  int           // valid after closed; 0 when there was no close frame
	closeText  string
}

// dial opens a socket on the node for the channel at the broker, and waits
// for the challenge message the node writes once the broker has accepted the
// join. The owner is the address the socket would publish as.
func dial(ctx context.Context, node *bee.Client, ch *channel, broker swarm.Address, owner []byte) (*socket, error) {
	q := url.Values{}
	q.Set("broker", broker.String())
	q.Set("owner", hex.EncodeToString(owner))
	u := fmt.Sprintf("ws://%s/bps/feeds/subscribe/%x/%x?%s", node.Host(), ch.principal, ch.topic, q.Encode())

	dialer := &websocket.Dialer{HandshakeTimeout: dialTimeout}
	ws, resp, err := dialer.DialContext(ctx, u, http.Header{})
	if err != nil {
		if resp != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("node %s: dial: %w (http %d)", node.Name(), err, resp.StatusCode)
		}
		return nil, fmt.Errorf("node %s: dial: %w", node.Name(), err)
	}
	s := &socket{
		node:       node.Name(),
		ws:         ws,
		challenges: make(chan string, 1),
		deliveries: make(chan delivery, maxMessages),
		closed:     make(chan struct{}),
	}
	go s.readLoop()

	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	select {
	case txt := <-s.challenges:
		if err := s.setChallenge(ch, txt); err != nil {
			s.close()
			return nil, fmt.Errorf("node %s: %w", s.node, err)
		}
		return s, nil
	case <-s.closed:
		return nil, &closedError{node: s.node, code: s.closeCode, text: s.closeText}
	case <-ctx.Done():
		s.close()
		return nil, fmt.Errorf("node %s: no challenge message: %w", s.node, ctx.Err())
	}
}

// setChallenge reads the challenge message and checks that the session topics
// the node reports are the ones the check derives.
func (s *socket) setChallenge(ch *channel, txt string) error {
	var m struct {
		Challenge string `json:"challenge"`
		DataTopic string `json:"dataTopic"`
		AuthTopic string `json:"authTopic"`
	}
	if err := json.Unmarshal([]byte(txt), &m); err != nil {
		return fmt.Errorf("challenge message %q: %w", txt, err)
	}
	challenge, err := hex.DecodeString(m.Challenge)
	if err != nil || len(challenge) != challengeSize {
		return fmt.Errorf("challenge %q is not %d bytes of hex", m.Challenge, challengeSize)
	}
	for kind, got := range map[byte]string{kindData: m.DataTopic, kindAuth: m.AuthTopic} {
		want, err := sessionTopic(kind, ch.topic, challenge)
		if err != nil {
			return err
		}
		if got != hex.EncodeToString(want) {
			return fmt.Errorf("session topic of kind %d is %s, derived %x", kind, got, want)
		}
	}
	s.challenge = challenge
	return nil
}

func (s *socket) readLoop() {
	defer close(s.closed)
	for {
		typ, data, err := s.ws.ReadMessage()
		if err != nil {
			var ce *websocket.CloseError
			if errors.As(err, &ce) {
				s.closeCode, s.closeText = ce.Code, ce.Text
			} else {
				s.closeText = err.Error()
			}
			return
		}
		switch typ {
		case websocket.TextMessage:
			select {
			case s.challenges <- string(data):
			default:
			}
		case websocket.BinaryMessage:
			if len(data) < deliveryHeader {
				s.closeText = fmt.Sprintf("delivery frame of %d bytes", len(data))
				_ = s.ws.Close()
				return
			}
			d := delivery{
				kind:      data[0],
				owner:     data[1 : 1+ownerSize],
				challenge: data[1+ownerSize : 1+ownerSize+challengeSize],
				index:     binary.BigEndian.Uint64(data[1+ownerSize+challengeSize : deliveryHeader]),
				soc:       data[deliveryHeader:],
			}
			select {
			case s.deliveries <- d:
			default:
				s.closeText = "more deliveries than the check reads"
				_ = s.ws.Close()
				return
			}
		}
	}
}

// publish signs a chunk of the kind at the index under the socket's challenge
// and sends it as one frame: kind | index | soc.
func (s *socket) publish(ch *channel, kind byte, index uint64, payload []byte) error {
	id, err := chunkID(kind, ch.topic, s.challenge, index)
	if err != nil {
		return err
	}
	c, err := cac.New(payload)
	if err != nil {
		return err
	}
	sch, err := soc.New(id, c).Sign(ch.signer)
	if err != nil {
		return err
	}
	frame := make([]byte, 0, 9+len(sch.Data()))
	frame = append(frame, kind)
	frame = binary.BigEndian.AppendUint64(frame, index)
	frame = append(frame, sch.Data()...)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.ws.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("node %s: write: %w", s.node, err)
	}
	return nil
}

// expect waits for the next delivery and requires it to be the principal's
// update at the index, with the payload, under the challenge.
func (s *socket) expect(ctx context.Context, ch *channel, challenge []byte, index uint64, payload []byte) error {
	select {
	case d := <-s.deliveries:
		got, err := d.payload(ch)
		if err != nil {
			return fmt.Errorf("node %s: delivery at index %d: %w", s.node, d.index, err)
		}
		if d.index != index || !bytes.Equal(got, payload) {
			return fmt.Errorf("node %s: delivery at index %d with payload %q, want index %d with %q", s.node, d.index, got, index, payload)
		}
		if !bytes.Equal(d.challenge, challenge) {
			return fmt.Errorf("node %s: delivery at index %d under challenge %x, want the publisher's %x", s.node, d.index, d.challenge, challenge)
		}
		return nil
	case <-s.closed:
		return &closedError{node: s.node, code: s.closeCode, text: s.closeText}
	case <-ctx.Done():
		return fmt.Errorf("node %s: no delivery at index %d: %w", s.node, index, ctx.Err())
	}
}

// close ends the socket with a normal close frame and waits for the read loop.
func (s *socket) close() {
	s.writeMu.Lock()
	_ = s.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	s.writeMu.Unlock()
	_ = s.ws.Close()
	<-s.closed
}
