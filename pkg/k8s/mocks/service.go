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
var _ corev1.ServiceInterface = (*Service)(nil)

type Service struct {
	corev1.ServiceInterface
}

func NewService() *Service {
	return &Service{}
}

// Create implements v1.ServiceInterface
func (*Service) Create(ctx context.Context, service *v1.Service, opts metav1.CreateOptions) (*v1.Service, error) {
	if service.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create service")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.ServiceInterface
func (*Service) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete service")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// Get implements v1.ServiceInterface
func (*Service) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Service, error) {
	switch name {
	case CreateBad:
		return nil, errors.NewNotFound(schema.GroupResource{}, name)
	case UpdateBad:
		return &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}, nil
	}
	return nil, fmt.Errorf("mock error: unknown")
}

// Update implements v1.ServiceInterface
func (*Service) Update(ctx context.Context, service *v1.Service, opts metav1.UpdateOptions) (*v1.Service, error) {
	if service.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update service")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, service.Name)
	}
}
