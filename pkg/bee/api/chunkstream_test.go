package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	pb "github.com/ethersphere/beekeeper/pkg/bee/api/chunkstreampb"
	"github.com/gorilla/websocket"
)

// chunkStreamServer answers every Put with STATUS_OK and every Get with the
// requested address as data, in reverse order of arrival once n requests are in.
func chunkStreamServer(t *testing.T, n int, gotBatch chan<- string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chunks/stream" {
			http.NotFound(w, r)
			return
		}
		gotBatch <- r.Header.Get(postageStampBatchHeader)

		upgrader := websocket.Upgrader{Subprotocols: []string{ChunkStreamSubprotocol}}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		var resps []*pb.Response
		for range n {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				t.Error(err)
				return
			}
			req := new(pb.Request)
			if err := req.Unmarshal(msg); err != nil {
				t.Error(err)
				return
			}
			switch {
			case req.GetPut() != nil:
				p := req.GetPut()
				status := pb.Status_STATUS_OK
				if !bytes.Equal(p.BatchId, []byte{1, 2}) {
					status = pb.Status_STATUS_BAD_REQUEST
				}
				resps = append(resps, &pb.Response{Body: &pb.Response_Put{Put: &pb.PutResponse{Address: p.Address, Status: status}}})
			case req.GetGet() != nil:
				g := req.GetGet()
				status := pb.Status_STATUS_OK
				if g.Cache != pb.CacheOption_CACHE_DISABLE {
					status = pb.Status_STATUS_BAD_REQUEST
				}
				resps = append(resps, &pb.Response{Body: &pb.Response_Get{Get: &pb.GetResponse{Address: g.Address, Status: status, Data: g.Address}}})
			}
		}
		for i := len(resps) - 1; i >= 0; i-- {
			msg, err := resps[i].Marshal()
			if err != nil {
				t.Error(err)
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				t.Error(err)
				return
			}
		}
		// Wait for the client's close frame.
		_, _, _ = conn.ReadMessage()
	}))
}

func TestChunkStream(t *testing.T) {
	gotBatch := make(chan string, 1)
	srv := chunkStreamServer(t, 2, gotBatch)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewClient(u, nil)
	if err != nil {
		t.Fatal(err)
	}

	stream, err := c.ChunkStream.Open(context.Background(), "abcd")
	if err != nil {
		t.Fatal(err)
	}
	if b := <-gotBatch; b != "abcd" {
		t.Fatalf("batch header: got %q, want %q", b, "abcd")
	}

	putAddr := swarm.MustParseHexAddress("aa00000000000000000000000000000000000000000000000000000000000000")
	getAddr := swarm.MustParseHexAddress("bb00000000000000000000000000000000000000000000000000000000000000")
	if err := stream.Put(putAddr, []byte("data"), []byte{1, 2}, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := stream.Get(getAddr, pb.CacheOption_CACHE_DISABLE, time.Second); err != nil {
		t.Fatal(err)
	}

	// The server replies in reverse order.
	resp, err := stream.Receive(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	g := resp.GetGet()
	if g == nil || g.Status != pb.Status_STATUS_OK || !bytes.Equal(g.Address, getAddr.Bytes()) || !bytes.Equal(g.Data, getAddr.Bytes()) {
		t.Fatalf("get reply: got %v", resp)
	}

	resp, err = stream.Receive(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	p := resp.GetPut()
	if p == nil || p.Status != pb.Status_STATUS_OK || !bytes.Equal(p.Address, putAddr.Bytes()) {
		t.Fatalf("put reply: got %v", resp)
	}

	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestChunkStreamConcurrentPuts puts from several goroutines on one stream and
// expects every request to reach the server intact (run with -race).
func TestChunkStreamConcurrentPuts(t *testing.T) {
	const n = 32

	gotBatch := make(chan string, 1)
	srv := chunkStreamServer(t, n, gotBatch)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewClient(u, nil)
	if err != nil {
		t.Fatal(err)
	}

	stream, err := c.ChunkStream.Open(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	<-gotBatch

	want := make(map[string]struct{}, n)
	addrs := make([]swarm.Address, n)
	for i := range addrs {
		b := make([]byte, swarm.HashSize)
		b[0], b[1] = byte(i), 0xcc
		addrs[i] = swarm.NewAddress(b)
		want[addrs[i].ByteString()] = struct{}{}
	}

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for _, addr := range addrs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- stream.Put(addr, bytes.Repeat([]byte{0xdd}, 4096), []byte{1, 2}, time.Second)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	for range n {
		resp, err := stream.Receive(time.Second)
		if err != nil {
			t.Fatal(err)
		}
		p := resp.GetPut()
		if p == nil || p.Status != pb.Status_STATUS_OK {
			t.Fatalf("put reply: got %v", resp)
		}
		if _, ok := want[string(p.Address)]; !ok {
			t.Fatalf("unexpected or duplicate reply for %x", p.Address)
		}
		delete(want, string(p.Address))
	}

	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestChunkStreamNoBatchHeader(t *testing.T) {
	gotBatch := make(chan string, 1)
	srv := chunkStreamServer(t, 0, gotBatch)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewClient(u, nil)
	if err != nil {
		t.Fatal(err)
	}

	stream, err := c.ChunkStream.Open(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if b := <-gotBatch; b != "" {
		t.Fatalf("batch header: got %q, want none", b)
	}
}

// TestChunkStreamOlderNode connects to a node that serves only the older
// upload-only stream, which selects no subprotocol, and expects Open to refuse
// it before any request is sent.
func TestChunkStreamOlderNode(t *testing.T) {
	gotMessage := make(chan bool, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		mt, _, err := conn.ReadMessage()
		gotMessage <- err == nil && mt == websocket.BinaryMessage
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewClient(u, nil)
	if err != nil {
		t.Fatal(err)
	}

	stream, err := c.ChunkStream.Open(context.Background(), "")
	if !errors.Is(err, ErrChunkStreamUnsupported) {
		if stream != nil {
			_ = stream.Close()
		}
		t.Fatalf("got error %v, want %v", err, ErrChunkStreamUnsupported)
	}
	select {
	case got := <-gotMessage:
		if got {
			t.Fatal("a request was sent to a node without the chunk stream protocol")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("connection was not closed")
	}
}

// TestChunkStreamRefusedHandshake connects to a node that refuses the upgrade
// and expects Open to report the node's status and message.
func TestChunkStreamRefusedHandshake(t *testing.T) {
	const message = "batch not usable"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":400,"message":"` + message + `"}`))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewClient(u, nil)
	if err != nil {
		t.Fatal(err)
	}

	stream, err := c.ChunkStream.Open(context.Background(), "")
	if err == nil {
		_ = stream.Close()
		t.Fatal("got no error, want a refused handshake")
	}
	if !errors.Is(err, websocket.ErrBadHandshake) {
		t.Fatalf("got error %v, want %v", err, websocket.ErrBadHandshake)
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.Code != http.StatusBadRequest {
		t.Fatalf("got error %v, want status %d", err, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("got error %q, want it to contain %q", err, message)
	}
}

// TestChunkStreamContextCancel cancels the context of an open stream and
// expects a blocked Receive to return the context's error at once rather than
// at its deadline, and later calls to fail the same way.
func TestChunkStreamContextCancel(t *testing.T) {
	gotBatch := make(chan string, 1)
	srv := chunkStreamServer(t, 0, gotBatch)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewClient(u, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := c.ChunkStream.Open(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	<-gotBatch

	errc := make(chan error, 1)
	go func() {
		_, err := stream.Receive(time.Minute)
		errc <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("receive: got %v, want %v", err, context.Canceled)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("receive still blocked after the context was canceled")
	}

	addr := swarm.MustParseHexAddress("aa00000000000000000000000000000000000000000000000000000000000000")
	if err := stream.Put(addr, []byte("data"), nil, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("put: got %v, want %v", err, context.Canceled)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
