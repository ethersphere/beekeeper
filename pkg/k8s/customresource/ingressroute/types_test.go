package ingressroute_test

import (
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressRouteDeepCopy(t *testing.T) {
	t.Parallel()
	src := ingressroute.IngressRoute{
		TypeMeta:   metav1.TypeMeta{Kind: "IngressRoute", APIVersion: ingressroute.GroupName},
		ObjectMeta: metav1.ObjectMeta{Name: testIRName, Namespace: "test", Labels: map[string]string{"a": "b"}},
		Spec:       ingressroute.IngressRouteSpec{Routes: []ingressroute.Route{{Kind: "Rule", Match: `Host("x.example.com")`}}},
	}

	t.Run("DeepCopyObject", func(t *testing.T) {
		obj := src.DeepCopyObject()
		cp, ok := obj.(*ingressroute.IngressRoute)
		if !ok {
			t.Fatalf("expected *IngressRoute, got %T", obj)
		}
		if cp == &src {
			t.Error("DeepCopyObject returned the original pointer")
		}
		if !reflect.DeepEqual(*cp, src) {
			t.Errorf("copy not equal:\n got: %#v\nwant: %#v", *cp, src)
		}
	})

	t.Run("DeepCopyInto", func(t *testing.T) {
		var dst ingressroute.IngressRoute
		src.DeepCopyInto(&dst)
		if !reflect.DeepEqual(dst, src) {
			t.Errorf("DeepCopyInto mismatch:\n got: %#v\nwant: %#v", dst, src)
		}
	})
}

func TestIngressRouteListDeepCopyObject(t *testing.T) {
	t.Parallel()
	t.Run("with_items", func(t *testing.T) {
		src := ingressroute.IngressRouteList{
			TypeMeta: metav1.TypeMeta{Kind: "IngressRouteList"},
			Items: []ingressroute.IngressRoute{
				{ObjectMeta: metav1.ObjectMeta{Name: testIRName, Namespace: "test"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ir-1", Namespace: "test"}},
			},
		}
		obj := src.DeepCopyObject()
		cp, ok := obj.(*ingressroute.IngressRouteList)
		if !ok {
			t.Fatalf("expected *IngressRouteList, got %T", obj)
		}
		if !reflect.DeepEqual(*cp, src) {
			t.Errorf("copy not equal:\n got: %#v\nwant: %#v", *cp, src)
		}
		if len(cp.Items) != 2 {
			t.Errorf("expected 2 items, got: %d", len(cp.Items))
		}
	})

	t.Run("nil_items", func(t *testing.T) {
		src := ingressroute.IngressRouteList{TypeMeta: metav1.TypeMeta{Kind: "IngressRouteList"}}
		cp, ok := src.DeepCopyObject().(*ingressroute.IngressRouteList)
		if !ok {
			t.Fatalf("expected *IngressRouteList")
		}
		if cp.Items != nil {
			t.Errorf("expected nil items, got: %#v", cp.Items)
		}
	})
}
