package ingressroute_test

import (
	"errors"
	"io"
	"reflect"
	"sort"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute/mock"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/logging"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// testIRName is the canonical IngressRoute name used across the ingressroute
// tests.
const testIRName = "ir-0"

func newClient(opts ...mock.Option) *ingressroute.Client {
	return ingressroute.NewClient(mock.New(opts...), logging.New(io.Discard, 0))
}

// newIR builds an IngressRoute in namespace "test" with one Route per match.
func newIR(name string, matches ...string) ingressroute.IngressRoute {
	ir := ingressroute.IngressRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"},
	}
	for _, m := range matches {
		ir.Spec.Routes = append(ir.Spec.Routes, ingressroute.Route{Match: m})
	}
	return ir
}

func TestClientSet(t *testing.T) {
	t.Parallel()
	t.Run("create_when_not_found", func(t *testing.T) {
		client := newClient()
		ing, err := client.Set(t.Context(), testIRName, "test", ingressroute.Options{})
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if ing == nil || ing.Name != testIRName {
			t.Errorf("expected created ingress route ir-0, got: %#v", ing)
		}
	})

	t.Run("update_when_found", func(t *testing.T) {
		client := newClient(mock.WithIngressRoutes(newIR(testIRName)))
		ing, err := client.Set(t.Context(), testIRName, "test", ingressroute.Options{})
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if ing == nil || ing.Name != testIRName {
			t.Errorf("expected updated ingress route ir-0, got: %#v", ing)
		}
	})

	t.Run("get_error", func(t *testing.T) {
		client := newClient(mock.WithGetError(errors.New("mock error")))
		_, err := client.Set(t.Context(), testIRName, "test", ingressroute.Options{})
		if err == nil || err.Error() != "getting ingress route ir-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("create_error", func(t *testing.T) {
		client := newClient(mock.WithCreateError(errors.New("mock error")))
		_, err := client.Set(t.Context(), testIRName, "test", ingressroute.Options{})
		if err == nil || err.Error() != "creating ingress route ir-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("update_error", func(t *testing.T) {
		client := newClient(mock.WithIngressRoutes(newIR(testIRName)), mock.WithUpdateError(errors.New("mock error")))
		_, err := client.Set(t.Context(), testIRName, "test", ingressroute.Options{})
		if err == nil || err.Error() != "updating ingress route ir-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestClientDelete(t *testing.T) {
	t.Parallel()
	t.Run("delete_existing", func(t *testing.T) {
		client := newClient(mock.WithIngressRoutes(newIR(testIRName)))
		if err := client.Delete(t.Context(), testIRName, "test"); err != nil {
			t.Errorf("error not expected, got: %s", err.Error())
		}
	})

	t.Run("delete_not_found_is_nil", func(t *testing.T) {
		client := newClient()
		if err := client.Delete(t.Context(), testIRName, "test"); err != nil {
			t.Errorf("not-found delete should be nil, got: %s", err.Error())
		}
	})

	t.Run("delete_error", func(t *testing.T) {
		client := newClient(mock.WithDeleteError(errors.New("mock error")))
		err := client.Delete(t.Context(), testIRName, "test")
		if err == nil || err.Error() != "deleting ingress route ir-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestClientGetNodes(t *testing.T) {
	t.Parallel()
	sortNodes := func(nodes []ingress.NodeInfo) {
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].Name != nodes[j].Name {
				return nodes[i].Name < nodes[j].Name
			}
			return nodes[i].Host < nodes[j].Host
		})
	}

	t.Run("extracts_hosts", func(t *testing.T) {
		client := newClient(mock.WithIngressRoutes(
			// the PathPrefix route has no Host(...) → GetHost returns "" → skipped
			newIR(testIRName, `Host("a.example.com")`, "PathPrefix(`/x`)"),
			newIR("ir-1", `Host("b.example.com")`),
		))
		nodes, err := client.GetNodes(t.Context(), "test", "")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		sortNodes(nodes)
		expected := []ingress.NodeInfo{
			{Name: testIRName, Host: "a.example.com"},
			{Name: "ir-1", Host: "b.example.com"},
		}
		if !reflect.DeepEqual(nodes, expected) {
			t.Errorf("nodes expected: %#v, got: %#v", expected, nodes)
		}
	})

	t.Run("no_routes", func(t *testing.T) {
		client := newClient()
		nodes, err := client.GetNodes(t.Context(), "test", "")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if nodes != nil {
			t.Errorf("nodes expected nil, got: %#v", nodes)
		}
	})

	t.Run("list_not_found_is_nil", func(t *testing.T) {
		client := newClient(mock.WithListError(apierrors.NewNotFound(schema.GroupResource{}, "")))
		nodes, err := client.GetNodes(t.Context(), "test", "")
		if err != nil {
			t.Errorf("not-found list should be nil error, got: %s", err.Error())
		}
		if nodes != nil {
			t.Errorf("nodes expected nil, got: %#v", nodes)
		}
	})

	t.Run("list_error", func(t *testing.T) {
		client := newClient(mock.WithListError(errors.New("mock error")))
		_, err := client.GetNodes(t.Context(), "test", "")
		if err == nil || err.Error() != "list ingress routes in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
