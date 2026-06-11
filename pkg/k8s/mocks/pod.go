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
var _ corev1.PodInterface = (*Pod)(nil)

type Pod struct {
	corev1.PodInterface
}

func NewPod() *Pod {
	return &Pod{}
}

// Create implements v1.PodInterface
func (*Pod) Create(ctx context.Context, pod *v1.Pod, opts metav1.CreateOptions) (*v1.Pod, error) {
	if pod.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create pod")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.PodInterface
func (*Pod) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete pod")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// Update implements v1.PodInterface
func (*Pod) Update(ctx context.Context, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	if pod.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update pod")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, pod.Name)
	}
}
