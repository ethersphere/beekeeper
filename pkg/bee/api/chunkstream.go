package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	pb "github.com/ethersphere/beekeeper/pkg/bee/api/chunkstreampb"
	"github.com/gorilla/websocket"
)

// ChunkStreamService represents Bee's /chunks/stream websocket endpoint, which
// carries chunk downloads (Get) and uploads (Put) over a single connection.
type ChunkStreamService service

const chunkStreamHandshakeTimeout = 45 * time.Second

// ChunkStream is an open /chunks/stream connection. Requests can be pipelined:
// the node answers each one with exactly one response carrying the same
// address, in no particular order. It is not safe for concurrent writes or
// concurrent reads.
type ChunkStream struct {
	conn *websocket.Conn
}

// Open opens a chunk stream. When batchID is not empty it is sent as
// Swarm-Postage-Batch-Id, which stamps every Put on the connection with that
// batch and pins the connection to it.
func (c *ChunkStreamService) Open(ctx context.Context, batchID string) (*ChunkStream, error) {
	header := http.Header{}
	if batchID != "" {
		header.Set(postageStampBatchHeader, batchID)
	}

	full, err := c.client.getFullURL("/" + apiVersion + "/chunks/stream")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(full)
	if err != nil {
		return nil, fmt.Errorf("parse websocket url: %w", err)
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: chunkStreamHandshakeTimeout,
	}
	conn, resp, err := dialer.DialContext(ctx, u.String(), header)
	if resp != nil {
		defer drain(resp.Body)
	}
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("dial %s: %w (%s)", u, err, resp.Status)
		}
		return nil, fmt.Errorf("dial %s: %w", u, err)
	}

	return &ChunkStream{conn: conn}, nil
}

// Get requests the chunk at addr.
func (s *ChunkStream) Get(addr swarm.Address, cache pb.CacheOption, timeout time.Duration) error {
	return s.send(&pb.Request{Body: &pb.Request_Get{Get: &pb.GetRequest{
		Address: addr.Bytes(),
		Cache:   cache,
	}}}, timeout)
}

// Put uploads a chunk, a CAC or a SOC, under addr. When batchID is set, the node
// stamps the chunk with that batch instead of the connection's.
func (s *ChunkStream) Put(addr swarm.Address, data, batchID []byte, timeout time.Duration) error {
	return s.send(&pb.Request{Body: &pb.Request_Put{Put: &pb.PutRequest{
		Address: addr.Bytes(),
		Data:    data,
		BatchId: batchID,
	}}}, timeout)
}

func (s *ChunkStream) send(req *pb.Request, timeout time.Duration) error {
	msg, err := req.Marshal()
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if err := s.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	return nil
}

// Receive reads the next response.
func (s *ChunkStream) Receive(timeout time.Duration) (*pb.Response, error) {
	if err := s.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}
	mt, msg, err := s.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if mt != websocket.BinaryMessage {
		return nil, fmt.Errorf("response: expected a binary frame, got message type %d", mt)
	}

	resp := new(pb.Response)
	if err := resp.Unmarshal(msg); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if resp.GetGet() == nil && resp.GetPut() == nil {
		return nil, errors.New("response has no body")
	}

	return resp, nil
}

// Close sends a normal close frame and closes the connection.
func (s *ChunkStream) Close() error {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	if err := s.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(5*time.Second)); err != nil &&
		!errors.Is(err, websocket.ErrCloseSent) {
		_ = s.conn.Close()
		return fmt.Errorf("send close message: %w", err)
	}

	return s.conn.Close()
}
