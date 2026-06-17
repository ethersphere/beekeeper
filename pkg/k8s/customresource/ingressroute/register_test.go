package ingressroute_test

import (
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestKind(t *testing.T) {
	t.Parallel()
	gk := ingressroute.Kind("IngressRoute")
	if gk.Group != ingressroute.GroupName || gk.Kind != "IngressRoute" {
		t.Errorf("unexpected GroupKind: %#v", gk)
	}
}

func TestResource(t *testing.T) {
	t.Parallel()
	gr := ingressroute.Resource("ingressroutes")
	if gr.Group != ingressroute.GroupName || gr.Resource != "ingressroutes" {
		t.Errorf("unexpected GroupResource: %#v", gr)
	}
}

// TestAddToScheme covers addKnownTypes via the exported AddToScheme builder.
func TestAddToScheme(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	if err := ingressroute.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme failed: %s", err.Error())
	}
	for _, kind := range []string{"IngressRoute", "IngressRouteList"} {
		gvk := ingressroute.SchemeGroupVersion.WithKind(kind)
		if !scheme.Recognizes(gvk) {
			t.Errorf("scheme does not recognize %s", gvk)
		}
	}
}
