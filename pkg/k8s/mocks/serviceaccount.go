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
var _ corev1.ServiceAccountInterface = (*ServiceAccount)(nil)

type ServiceAccount struct {
	corev1.ServiceAccountInterface
}

func NewServiceAccount() *ServiceAccount {
	return &ServiceAccount{}
}

// Create implements v1.ServiceAccountInterface
func (*ServiceAccount) Create(ctx context.Context, serviceAccount *v1.ServiceAccount, opts metav1.CreateOptions) (*v1.ServiceAccount, error) {
	if serviceAccount.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create service account")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.ServiceAccountInterface
func (*ServiceAccount) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete service account")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// Update implements v1.ServiceAccountInterface
func (*ServiceAccount) Update(ctx context.Context, serviceAccount *v1.ServiceAccount, opts metav1.UpdateOptions) (*v1.ServiceAccount, error) {
	if serviceAccount.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update service account")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, serviceAccount.Name)
	}
}
