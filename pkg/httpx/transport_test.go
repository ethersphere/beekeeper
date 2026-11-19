package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/httpx"
)

func TestHeaderRoundTripper(t *testing.T) {
	gotUA := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA <- r.Header.Get("User-Agent")
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "caller")

	// Next is nil, so the request goes through http.DefaultTransport.
	resp, err := (&httpx.HeaderRoundTripper{}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if ua := <-gotUA; !strings.HasPrefix(ua, "beekeeper/") {
		t.Errorf("server got User-Agent %q, want beekeeper/ prefix", ua)
	}
	if ua := req.Header.Get("User-Agent"); ua != "caller" {
		t.Errorf("original request User-Agent changed to %q", ua)
	}
}

func TestNewClient_TLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)

	tests := []struct {
		name    string
		opts    httpx.ClientOptions
		wantErr bool
	}{
		{name: "verifies certificates by default", wantErr: true},
		{name: "insecure skips verification", opts: httpx.ClientOptions{InsecureSkipVerify: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := httpx.NewClient(tt.opts)
			t.Cleanup(c.CloseIdleConnections)

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := c.Do(req)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("got error %v, want error %t", err, tt.wantErr)
			}
			if resp != nil {
				_ = resp.Body.Close()
			}
		})
	}
}
