package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/gorilla/websocket"
)

// ChunkStreamService represents Bee's chunk stream service, a websocket
// endpoint that multiplexes many chunk uploads or downloads over a single
// connection.
type ChunkStreamService service

// Delivery status reported by the download stream for each requested address.
const (
	ChunkDeliverySuccess  byte = 0x00
	ChunkDeliveryNotFound byte = 0x01
	ChunkDeliveryError    byte = 0x02
)

const (
	chunkStreamUploadSubprotocol   = "swarm-chunk-upload"
	chunkStreamDownloadSubprotocol = "swarm-chunk-download"
	chunkStreamHandshakeTimeout    = 45 * time.Second

	// chunkDownloadOpcode is the command byte that prefixes a download request
	// frame: [opcode][32-byte address]...
	chunkDownloadOpcode byte = 'D'
)

func chunkStreamPath() string {
	return "/" + apiVersion + "/chunks/stream"
}

// ChunkDelivery is a single response frame from the download stream. Data is
// only set when Status is ChunkDeliverySuccess.
type ChunkDelivery struct {
	Status  byte
	Address swarm.Address
	Data    []byte
}

// ChunkUploadStream uploads chunks over a single websocket connection. It is
// not safe for concurrent use: each Upload writes a chunk and waits for the
// node to acknowledge it.
type ChunkUploadStream struct {
	conn *websocket.Conn
}

// NewUploadStream opens a chunk stream in upload mode. Chunks are stamped with
// the given postage batch and pushed to the network as they arrive.
func (c *ChunkStreamService) NewUploadStream(ctx context.Context, batchID string) (*ChunkUploadStream, error) {
	header := http.Header{}
	header.Set(postageStampBatchHeader, batchID)

	conn, err := c.client.dialWebSocket(ctx, chunkStreamPath(), chunkStreamUploadSubprotocol, header)
	if err != nil {
		return nil, err
	}

	return &ChunkUploadStream{conn: conn}, nil
}

// Upload writes one chunk and waits for the node's acknowledgement, which the
// node sends as an empty binary frame.
func (s *ChunkUploadStream) Upload(data []byte, timeout time.Duration) error {
	if err := s.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if err := s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("write chunk: %w", err)
	}

	if err := s.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}
	mt, msg, err := s.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read acknowledgement: %w", err)
	}
	if mt != websocket.BinaryMessage {
		return fmt.Errorf("acknowledgement: expected a binary frame, got message type %d", mt)
	}
	if len(msg) != 0 {
		return fmt.Errorf("acknowledgement: expected an empty frame, got %d bytes", len(msg))
	}

	return nil
}

// Close shuts the stream down, giving the node a chance to finish syncing.
func (s *ChunkUploadStream) Close() error {
	return closeWebSocket(s.conn)
}

// ChunkDownloadStream requests chunks over a single websocket connection.
// Requests are pipelined and the node answers out of order, so callers must
// match deliveries by address rather than by request order.
type ChunkDownloadStream struct {
	conn *websocket.Conn
}

// NewDownloadStream opens a chunk stream in download mode.
func (c *ChunkStreamService) NewDownloadStream(ctx context.Context) (*ChunkDownloadStream, error) {
	conn, err := c.client.dialWebSocket(ctx, chunkStreamPath(), chunkStreamDownloadSubprotocol, http.Header{})
	if err != nil {
		return nil, err
	}

	return &ChunkDownloadStream{conn: conn}, nil
}

// Request asks for the given addresses in a single frame, sent as
// [opcode][32-byte address]... The node answers with exactly one delivery per
// address, in no particular order.
func (s *ChunkDownloadStream) Request(addrs []swarm.Address, timeout time.Duration) error {
	if len(addrs) == 0 {
		return errors.New("no addresses requested")
	}

	frame := make([]byte, 0, 1+len(addrs)*swarm.HashSize)
	frame = append(frame, chunkDownloadOpcode)
	for _, a := range addrs {
		if len(a.Bytes()) != swarm.HashSize {
			return fmt.Errorf("address %s is not %d bytes", a, swarm.HashSize)
		}
		frame = append(frame, a.Bytes()...)
	}

	if err := s.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("request %d addresses: %w", len(addrs), err)
	}

	return nil
}

// Receive reads the next delivery from the stream.
func (s *ChunkDownloadStream) Receive(timeout time.Duration) (ChunkDelivery, error) {
	if err := s.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return ChunkDelivery{}, fmt.Errorf("set read deadline: %w", err)
	}

	mt, msg, err := s.conn.ReadMessage()
	if err != nil {
		return ChunkDelivery{}, fmt.Errorf("read delivery: %w", err)
	}
	if mt != websocket.BinaryMessage {
		return ChunkDelivery{}, fmt.Errorf("delivery: expected a binary frame, got message type %d", mt)
	}
	if len(msg) < 1+swarm.HashSize {
		return ChunkDelivery{}, fmt.Errorf("delivery: frame of %d bytes is shorter than the %d byte header", len(msg), 1+swarm.HashSize)
	}

	d := ChunkDelivery{
		Status:  msg[0],
		Address: swarm.NewAddress(slices.Clone(msg[1 : 1+swarm.HashSize])),
	}
	if d.Status == ChunkDeliverySuccess {
		d.Data = slices.Clone(msg[1+swarm.HashSize:])
	}

	return d, nil
}

// Close shuts the stream down.
func (s *ChunkDownloadStream) Close() error {
	return closeWebSocket(s.conn)
}

// dialWebSocket opens a websocket connection to path on the node, negotiating
// the given subprotocol.
func (c *Client) dialWebSocket(ctx context.Context, path, subprotocol string, header http.Header) (*websocket.Conn, error) {
	full, err := c.getFullURL(path)
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
		Subprotocols:     []string{subprotocol},
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

	if got := resp.Header.Get("Sec-WebSocket-Protocol"); got != subprotocol {
		defer conn.Close()
		return nil, fmt.Errorf("node did not negotiate subprotocol %s, got %q", subprotocol, got)
	}

	return conn, nil
}

func closeWebSocket(conn *websocket.Conn) error {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	if err := conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(5*time.Second)); err != nil &&
		!errors.Is(err, websocket.ErrCloseSent) {
		// Best effort: still close the underlying connection below.
		_ = conn.Close()
		return fmt.Errorf("send close message: %w", err)
	}

	return conn.Close()
}
