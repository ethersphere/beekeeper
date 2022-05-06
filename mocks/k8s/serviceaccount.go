package mocks

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
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
var _ corev1.ServiceAccountInterface = (*ServiceAccount)(nil)

type ServiceAccount struct{}

func NewServiceAccount() *ServiceAccount {
	return &ServiceAccount{}
}

// Apply implements v1.ServiceAccountInterface
func (*ServiceAccount) Apply(ctx context.Context, serviceAccount *configcorev1.ServiceAccountApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ServiceAccount, err error) {
	panic("unimplemented")
}

// Create implements v1.ServiceAccountInterface
func (*ServiceAccount) Create(ctx context.Context, serviceAccount *v1.ServiceAccount, opts metav1.CreateOptions) (*v1.ServiceAccount, error) {
	if serviceAccount.ObjectMeta.Name == "create_bad" {
		return nil, fmt.Errorf("mock error: cannot create service account")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// CreateToken implements v1.ServiceAccountInterface
func (*ServiceAccount) CreateToken(ctx context.Context, serviceAccountName string, tokenRequest *authenticationv1.TokenRequest, opts metav1.CreateOptions) (*authenticationv1.TokenRequest, error) {
	panic("unimplemented")
}

// Delete implements v1.ServiceAccountInterface
func (*ServiceAccount) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete_bad" {
		return fmt.Errorf("mock error: cannot delete service account")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.ServiceAccountInterface
func (*ServiceAccount) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.ServiceAccountInterface
func (*ServiceAccount) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ServiceAccount, error) {
	panic("unimplemented")
}

// List implements v1.ServiceAccountInterface
func (*ServiceAccount) List(ctx context.Context, opts metav1.ListOptions) (*v1.ServiceAccountList, error) {
	panic("unimplemented")
}

// Patch implements v1.ServiceAccountInterface
func (*ServiceAccount) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ServiceAccount, err error) {
	panic("unimplemented")
}

// Update implements v1.ServiceAccountInterface
func (*ServiceAccount) Update(ctx context.Context, serviceAccount *v1.ServiceAccount, opts metav1.UpdateOptions) (*v1.ServiceAccount, error) {
	if serviceAccount.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error: cannot update service account")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, serviceAccount.ObjectMeta.Name)
	}
}

// Watch implements v1.ServiceAccountInterface
func (*ServiceAccount) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
