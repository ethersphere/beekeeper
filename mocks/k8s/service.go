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
	restclient "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.ServiceInterface = (*Service)(nil)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// ProxyGet implements v1.ServiceInterface
func (*Service) ProxyGet(scheme string, name string, port string, path string, params map[string]string) restclient.ResponseWrapper {
	panic("unimplemented")
}

// Apply implements v1.ServiceInterface
func (*Service) Apply(ctx context.Context, service *configcorev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Service, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.ServiceInterface
func (*Service) ApplyStatus(ctx context.Context, service *configcorev1.ServiceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Service, err error) {
	panic("unimplemented")
}

// Create implements v1.ServiceInterface
func (*Service) Create(ctx context.Context, service *v1.Service, opts metav1.CreateOptions) (*v1.Service, error) {
	if service.ObjectMeta.Name == "create_bad" {
		return nil, fmt.Errorf("mock error: cannot create service")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.ServiceInterface
func (*Service) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete_bad" {
		return fmt.Errorf("mock error: cannot delete service")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// Get implements v1.ServiceInterface
func (*Service) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Service, error) {
	if name == "create_bad" {
		return nil, errors.NewNotFound(schema.GroupResource{}, name)
	} else if name == "update_bad" {
		return &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}, nil
	}
	return nil, fmt.Errorf("mock error: unknown")
}

// List implements v1.ServiceInterface
func (*Service) List(ctx context.Context, opts metav1.ListOptions) (*v1.ServiceList, error) {
	panic("unimplemented")
}

// Patch implements v1.ServiceInterface
func (*Service) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Service, err error) {
	panic("unimplemented")
}

// Update implements v1.ServiceInterface
func (*Service) Update(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (*v1.Service, error) {
	if service.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error: cannot update service")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, service.ObjectMeta.Name)
	}
}

// UpdateStatus implements v1.ServiceInterface
func (*Service) UpdateStatus(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (*v1.Service, error) {
	panic("unimplemented")
}

// Watch implements v1.ServiceInterface
func (*Service) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
