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
var _ corev1.PersistentVolumeClaimInterface = (*Pvc)(nil)

type Pvc struct {
	corev1.PersistentVolumeClaimInterface
}

func NewPvc() *Pvc {
	return &Pvc{}
}

// Create implements v1.PersistentVolumeClaimInterface
func (*Pvc) Create(ctx context.Context, persistentVolumeClaim *v1.PersistentVolumeClaim, opts metav1.CreateOptions) (*v1.PersistentVolumeClaim, error) {
	if persistentVolumeClaim.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create pvc")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.PersistentVolumeClaimInterface
func (*Pvc) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete pvc")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// Update implements v1.PersistentVolumeClaimInterface
func (*Pvc) Update(ctx context.Context, persistentVolumeClaim *v1.PersistentVolumeClaim, opts metav1.UpdateOptions) (*v1.PersistentVolumeClaim, error) {
	if persistentVolumeClaim.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update pvc")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, persistentVolumeClaim.Name)
	}
}
