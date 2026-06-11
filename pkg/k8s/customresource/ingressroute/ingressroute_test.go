package ingressroute_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// TestRESTClient exercises the real REST layer (config.NewForConfig +
// ingressRouteClient CRUD/Watch) against an httptest.Server. httptest is used
// instead of client-go's fake RESTClient so the custom Traefik scheme is wired
// up by NewForConfig itself (no manual serializer/scheme scaffolding).
func TestRESTClient(t *testing.T) {
	t.Parallel()
	apiVersion := ingressroute.GroupName + "/" + ingressroute.GroupVersion

	ir := ingressroute.IngressRoute{
		TypeMeta:   metav1.TypeMeta{APIVersion: apiVersion, Kind: "IngressRoute"},
		ObjectMeta: metav1.ObjectMeta{Name: testIRName, Namespace: "test"},
		Spec:       ingressroute.IngressRouteSpec{Routes: []ingressroute.Route{{Match: `Host("x.example.com")`}}},
	}
	irList := ingressroute.IngressRouteList{
		TypeMeta: metav1.TypeMeta{APIVersion: apiVersion, Kind: "IngressRouteList"},
		Items:    []ingressroute.IngressRoute{ir},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete:
			_ = json.NewEncoder(w).Encode(&metav1.Status{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
				Status:   metav1.StatusSuccess,
			})
		case r.Method == http.MethodGet && r.URL.Query().Get("watch") == "true":
			w.WriteHeader(http.StatusOK) // empty watch stream
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/ingressroutes"):
			_ = json.NewEncoder(w).Encode(&irList)
		default: // GET by name, POST (create), PUT (update)
			_ = json.NewEncoder(w).Encode(&ir)
		}
	}))
	defer server.Close()

	client, err := ingressroute.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("NewForConfig: %s", err.Error())
	}
	irs := client.IngressRoutes("test")
	ctx := t.Context()

	t.Run("Get", func(t *testing.T) {
		got, err := irs.Get(ctx, testIRName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Get: %s", err.Error())
		}
		if got.Name != testIRName {
			t.Errorf("expected name ir-0, got: %q", got.Name)
		}
	})

	t.Run("List", func(t *testing.T) {
		list, err := irs.List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("List: %s", err.Error())
		}
		if len(list.Items) != 1 {
			t.Errorf("expected 1 item, got: %d", len(list.Items))
		}
	})

	t.Run("Create", func(t *testing.T) {
		got, err := irs.Create(ctx, &ir)
		if err != nil {
			t.Fatalf("Create: %s", err.Error())
		}
		if got.Name != testIRName {
			t.Errorf("expected name ir-0, got: %q", got.Name)
		}
	})

	t.Run("Update", func(t *testing.T) {
		got, err := irs.Update(ctx, &ir, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("Update: %s", err.Error())
		}
		if got.Name != testIRName {
			t.Errorf("expected name ir-0, got: %q", got.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := irs.Delete(ctx, testIRName, metav1.DeleteOptions{}); err != nil {
			t.Errorf("Delete: %s", err.Error())
		}
	})

	t.Run("Watch", func(t *testing.T) {
		watcher, err := irs.Watch(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("Watch: %s", err.Error())
		}
		watcher.Stop()
	})
}
