package mocks

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	configcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.SecretInterface = (*Secret)(nil)

type Secret struct{}

func NewSecret() *Secret {
	return &Secret{}
}

// Apply implements v1.SecretInterface
func (*Secret) Apply(ctx context.Context, secret *configcorev1.SecretApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Secret, err error) {
	panic("unimplemented")
}

// Create implements v1.SecretInterface
func (*Secret) Create(ctx context.Context, secret *v1.Secret, opts metav1.CreateOptions) (*v1.Secret, error) {
	if secret.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create secret")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.SecretInterface
func (*Secret) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete secret")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.SecretInterface
func (*Secret) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.SecretInterface
func (*Secret) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Secret, error) {
	panic("unimplemented")
}

// List implements v1.SecretInterface
func (*Secret) List(ctx context.Context, opts metav1.ListOptions) (*v1.SecretList, error) {
	panic("unimplemented")
}

// Patch implements v1.SecretInterface
func (*Secret) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Secret, err error) {
	panic("unimplemented")
}

// Update implements v1.SecretInterface
func (*Secret) Update(ctx context.Context, secret *v1.Secret, opts metav1.UpdateOptions) (*v1.Secret, error) {
	if secret.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update secret")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, secret.Name)
	}
}

// Watch implements v1.SecretInterface
func (*Secret) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
