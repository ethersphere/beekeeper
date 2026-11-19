package api_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	"github.com/ethersphere/beekeeper/pkg/bee/api"
	"github.com/ethersphere/beekeeper/pkg/httpx"
)

// newConnCountingServer starts a server that counts the TCP connections it
// accepts, so tests can tell whether the client reuses keep-alive connections.
func newConnCountingServer(t *testing.T, h http.Handler) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	var conns atomic.Int64
	srv := httptest.NewUnstartedServer(h)
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	return srv, &conns
}

func newAPIClient(t *testing.T, rawURL string, httpClient *http.Client) *api.Client {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}

	c, err := api.NewClient(u, httpClient)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestClient_ReusesConnections(t *testing.T) {
	const requests = 5

	// A body large enough that it is not consumed as a side effect of reading
	// the response headers, so the connection is only reusable if the client
	// drains and closes the body itself.
	largeBody := strings.Repeat("x", 16<<10)
	overlay := swarm.MustParseHexAddress("ca1e")

	tests := []struct {
		name    string
		handler http.HandlerFunc
		call    func(context.Context, *api.Client) error
		wantErr bool
	}{
		{
			name: "request with header, response not decoded",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(largeBody))
			},
			call: func(ctx context.Context, c *api.Client) error {
				return c.PSS.SendMessage(ctx, overlay, "key", "topic", 2, strings.NewReader("msg"), "batch")
			},
		},
		{
			name: "request with header, error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(largeBody))
			},
			call: func(ctx context.Context, c *api.Client) error {
				return c.PSS.SendMessage(ctx, overlay, "key", "topic", 2, strings.NewReader("msg"), "batch")
			},
			wantErr: true,
		},
		{
			name: "request data, error response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(largeBody))
			},
			call: func(ctx context.Context, c *api.Client) error {
				_, err := c.Bytes.Download(ctx, swarm.ZeroAddress, nil)
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, conns := newConnCountingServer(t, tt.handler)
			c := newAPIClient(t, srv.URL, srv.Client())

			for range requests {
				err := tt.call(t.Context(), c)
				if gotErr := err != nil; gotErr != tt.wantErr {
					t.Fatalf("got error %v, want error %t", err, tt.wantErr)
				}
			}

			if got := conns.Load(); got != 1 {
				t.Errorf("got %d connections for %d sequential requests, want 1", got, requests)
			}
		})
	}
}

func TestClient_SharedHTTPClientAcrossNodes(t *testing.T) {
	const requests = 3

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","version":"1"}`))
	})

	srvA, connsA := newConnCountingServer(t, handler)
	srvB, connsB := newConnCountingServer(t, handler)

	httpClient := httpx.NewClient(httpx.ClientOptions{})
	t.Cleanup(httpClient.CloseIdleConnections)

	nodes := []*api.Client{
		newAPIClient(t, srvA.URL, httpClient),
		newAPIClient(t, srvB.URL, httpClient),
	}

	for range requests {
		for _, n := range nodes {
			if _, err := n.Node.Health(t.Context()); err != nil {
				t.Fatalf("%s: %v", n.Host(), err)
			}
		}
	}

	for name, conns := range map[string]*atomic.Int64{"A": connsA, "B": connsB} {
		if got := conns.Load(); got != 1 {
			t.Errorf("node %s: got %d connections, want 1", name, got)
		}
	}
}

func TestClient_UploadSendsContentLength(t *testing.T) {
	const data = "swarm"

	tests := []struct {
		name   string
		upload func(context.Context, *api.Client, io.Reader, int64) error
	}{
		{
			name: "files",
			upload: func(ctx context.Context, c *api.Client, r io.Reader, size int64) error {
				_, err := c.Files.Upload(ctx, "file", r, size, api.UploadOptions{BatchID: "batch"})
				return err
			},
		},
		{
			name: "dirs",
			upload: func(ctx context.Context, c *api.Client, r io.Reader, size int64) error {
				_, err := c.Dirs.Upload(ctx, r, size, api.UploadOptions{BatchID: "batch"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type received struct {
				contentLength    int64
				transferEncoding []string
				body             string
			}
			got := make(chan received, 1)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				got <- received{contentLength: r.ContentLength, transferEncoding: r.TransferEncoding, body: string(b)}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"reference":"ca1e"}`))
			}))
			t.Cleanup(srv.Close)

			c := newAPIClient(t, srv.URL, srv.Client())

			// A reader whose length net/http cannot infer, as used by bee.Client.
			body := io.TeeReader(strings.NewReader(data), io.Discard)
			if err := tt.upload(t.Context(), c, body, int64(len(data))); err != nil {
				t.Fatal(err)
			}

			r := <-got
			if r.contentLength != int64(len(data)) {
				t.Errorf("got Content-Length %d, want %d", r.contentLength, len(data))
			}
			if len(r.transferEncoding) != 0 {
				t.Errorf("got Transfer-Encoding %v, want none", r.transferEncoding)
			}
			if r.body != data {
				t.Errorf("got body %q, want %q", r.body, data)
			}
		})
	}
}
