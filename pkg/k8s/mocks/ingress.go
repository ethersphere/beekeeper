package mocks

import (
	"context"
	"fmt"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.IngressInterface = (*Ingress)(nil)

type Ingress struct {
	v1.IngressInterface
}

func NewIngress() *Ingress {
	return &Ingress{}
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

// Update implements v1.IngressInterface
func (*Ingress) Update(ctx context.Context, ingress *netv1.Ingress, opts metav1.UpdateOptions) (*netv1.Ingress, error) {
	if ingress.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update ingress")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, ingress.Name)
	}
}
