package mocks

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ typedv1.NamespaceInterface = (*Namespace)(nil)

type Namespace struct {
	typedv1.NamespaceInterface
}

func NewNamespace() *Namespace {
	return &Namespace{}
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
