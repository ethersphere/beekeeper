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
var _ v1.IngressInterface = (*IngressMock)(nil)

type IngressMock struct{}

func NewIngressMock() *IngressMock {
	return &IngressMock{}
}

// Apply implements v1.IngressInterface
func (*IngressMock) Apply(ctx context.Context, ingress *networkingv1.IngressApplyConfiguration, opts metav1.ApplyOptions) (result *netv1.Ingress, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.IngressInterface
func (*IngressMock) ApplyStatus(ctx context.Context, ingress *networkingv1.IngressApplyConfiguration, opts metav1.ApplyOptions) (result *netv1.Ingress, err error) {
	panic("unimplemented")
}

// Create implements v1.IngressInterface
func (*IngressMock) Create(ctx context.Context, ingress *netv1.Ingress, opts metav1.CreateOptions) (*netv1.Ingress, error) {
	if ingress.ObjectMeta.Name == "create_bad" {
		return nil, fmt.Errorf("mock error: cannot create ingress")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.IngressInterface
func (*IngressMock) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete_bad" {
		return fmt.Errorf("mock error: cannot delete ingress")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.IngressInterface
func (*IngressMock) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.IngressInterface
func (*IngressMock) Get(ctx context.Context, name string, opts metav1.GetOptions) (*netv1.Ingress, error) {
	panic("unimplemented")
}

// List implements v1.IngressInterface
func (*IngressMock) List(ctx context.Context, opts metav1.ListOptions) (*netv1.IngressList, error) {
	panic("unimplemented")
}

// Patch implements v1.IngressInterface
func (*IngressMock) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *netv1.Ingress, err error) {
	panic("unimplemented")
}

// Update implements v1.IngressInterface
func (*IngressMock) Update(ctx context.Context, ingress *netv1.Ingress, opts metav1.UpdateOptions) (*netv1.Ingress, error) {
	if ingress.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error: cannot update ingress")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, ingress.ObjectMeta.Name)
	}
}

// UpdateStatus implements v1.IngressInterface
func (*IngressMock) UpdateStatus(ctx context.Context, ingress *netv1.Ingress, opts metav1.UpdateOptions) (*netv1.Ingress, error) {
	panic("unimplemented")
}

// Watch implements v1.IngressInterface
func (*IngressMock) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
