package mocks

import (
	"context"
	"fmt"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	networkingv1 "k8s.io/client-go/applyconfigurations/networking/v1"
	v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.IngressInterface = (*Ingress)(nil)

type Ingress struct{}

func NewIngress() *Ingress {
	return &Ingress{}
}

// Apply implements v1.IngressInterface
func (*Ingress) Apply(ctx context.Context, ingress *networkingv1.IngressApplyConfiguration, opts metav1.ApplyOptions) (result *netv1.Ingress, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.IngressInterface
func (*Ingress) ApplyStatus(ctx context.Context, ingress *networkingv1.IngressApplyConfiguration, opts metav1.ApplyOptions) (result *netv1.Ingress, err error) {
	panic("unimplemented")
}

// Create implements v1.IngressInterface
func (*Ingress) Create(ctx context.Context, ingress *netv1.Ingress, opts metav1.CreateOptions) (*netv1.Ingress, error) {
	if ingress.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create ingress")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.IngressInterface
func (*Ingress) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete ingress")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.IngressInterface
func (*Ingress) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.IngressInterface
func (*Ingress) Get(ctx context.Context, name string, opts metav1.GetOptions) (*netv1.Ingress, error) {
	panic("unimplemented")
}

// List implements v1.IngressInterface
func (*Ingress) List(ctx context.Context, opts metav1.ListOptions) (*netv1.IngressList, error) {
	panic("unimplemented")
}

// Patch implements v1.IngressInterface
func (*Ingress) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *netv1.Ingress, err error) {
	panic("unimplemented")
}

// Update implements v1.IngressInterface
func (*Ingress) Update(ctx context.Context, ingress *netv1.Ingress, opts metav1.UpdateOptions) (*netv1.Ingress, error) {
	if ingress.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update ingress")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, ingress.Name)
	}
}

// UpdateStatus implements v1.IngressInterface
func (*Ingress) UpdateStatus(ctx context.Context, ingress *netv1.Ingress, opts metav1.UpdateOptions) (*netv1.Ingress, error) {
	panic("unimplemented")
}

// Watch implements v1.IngressInterface
func (*Ingress) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
