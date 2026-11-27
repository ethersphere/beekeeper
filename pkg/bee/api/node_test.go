package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethersphere/bee/v2/pkg/swarm"
)

func TestNode_Blocklist(t *testing.T) {
	t.Parallel()

	const peer = "02285dba0d8c9a4a4e5d4b0d1cb1f4c1d5b0a1c2d3e4f5061728394a5b6c7d8e"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/blocklist" {
			t.Errorf("got %s %s, want GET /blocklist", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"peers":[{"address":"` + peer + `","fullNode":true,"reason":"bps: forged delivery","duration":864000}]}`))
	}))
	t.Cleanup(srv.Close)

	got, err := newAPIClient(t, srv.URL, srv.Client()).Node.Blocklist(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Peers) != 1 {
		t.Fatalf("got %d peers, want 1", len(got.Peers))
	}
	p := got.Peers[0]
	if !p.Address.Equal(swarm.MustParseHexAddress(peer)) || p.Reason != "bps: forged delivery" || p.Duration != 864000 {
		t.Fatalf("got %+v", p)
	}
}

func TestNode_Text(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/metrics":
			_, _ = w.Write([]byte("bee_bps_cohorts 2\n"))
		case "/debug/pprof/goroutine?debug=2":
			_, _ = w.Write([]byte("goroutine 1 [running]:\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	node := newAPIClient(t, srv.URL, srv.Client()).Node

	if got, err := node.Metrics(context.Background()); err != nil || got != "bee_bps_cohorts 2\n" {
		t.Fatalf("metrics: got %q, %v", got, err)
	}
	if got, err := node.Goroutines(context.Background()); err != nil || got != "goroutine 1 [running]:\n" {
		t.Fatalf("goroutines: got %q, %v", got, err)
	}
}
