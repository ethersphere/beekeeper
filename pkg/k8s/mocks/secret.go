package mocks

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.SecretInterface = (*Secret)(nil)

type Secret struct {
	corev1.SecretInterface
}

func NewSecret() *Secret {
	return &Secret{}
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

// Update implements v1.SecretInterface
func (*Secret) Update(ctx context.Context, secret *v1.Secret, opts metav1.UpdateOptions) (*v1.Secret, error) {
	if secret.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update secret")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, secret.Name)
	}
}
