package mocks

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ typedv1.NamespaceInterface = (*Namespace)(nil)

type Namespace struct{}

func NewNamespace() *Namespace {
	return &Namespace{}
}

// Finalize implements v1.NamespaceInterface
func (*Namespace) Finalize(ctx context.Context, item *v1.Namespace, opts metav1.UpdateOptions) (*v1.Namespace, error) {
	panic("unimplemented")
}

// Apply implements v1.NamespaceInterface
func (*Namespace) Apply(ctx context.Context, namespace *corev1.NamespaceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Namespace, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.NamespaceInterface
func (*Namespace) ApplyStatus(ctx context.Context, namespace *corev1.NamespaceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Namespace, err error) {
	panic("unimplemented")
}

// Create implements v1.NamespaceInterface
func (nm *Namespace) Create(ctx context.Context, namespace *v1.Namespace, opts metav1.CreateOptions) (*v1.Namespace, error) {
	return namespace, nil
}

// Delete implements v1.NamespaceInterface
func (*Namespace) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return fmt.Errorf("mock error: namespace \"%s\" can not be deleted", name)
}

// Get implements v1.NamespaceInterface
func (*Namespace) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Namespace, error) {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "beekeeper",
			},
		},
	}, nil
}

// List implements v1.NamespaceInterface
func (*Namespace) List(ctx context.Context, opts metav1.ListOptions) (*v1.NamespaceList, error) {
	panic("unimplemented")
}

// Patch implements v1.NamespaceInterface
func (*Namespace) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Namespace, err error) {
	panic("unimplemented")
}

// Update implements v1.NamespaceInterface
func (*Namespace) Update(ctx context.Context, namespace *v1.Namespace, opts metav1.UpdateOptions) (*v1.Namespace, error) {
	panic("unimplemented")
}

// UpdateStatus implements v1.NamespaceInterface
func (*Namespace) UpdateStatus(ctx context.Context, namespace *v1.Namespace, opts metav1.UpdateOptions) (*v1.Namespace, error) {
	panic("unimplemented")
}

// Watch implements v1.NamespaceInterface
func (*Namespace) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
