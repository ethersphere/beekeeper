package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
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
