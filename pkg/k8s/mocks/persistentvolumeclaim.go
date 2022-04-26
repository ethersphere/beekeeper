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
var _ corev1.PersistentVolumeClaimInterface = (*PvcMock)(nil)

type PvcMock struct{}

func NewPvcMock() *PvcMock {
	return &PvcMock{}
}

// Apply implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Apply(ctx context.Context, persistentVolumeClaim *configcorev1.PersistentVolumeClaimApplyConfiguration, opts metav1.ApplyOptions) (result *v1.PersistentVolumeClaim, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.PersistentVolumeClaimInterface
func (*PvcMock) ApplyStatus(ctx context.Context, persistentVolumeClaim *configcorev1.PersistentVolumeClaimApplyConfiguration, opts metav1.ApplyOptions) (result *v1.PersistentVolumeClaim, err error) {
	panic("unimplemented")
}

// Create implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Create(ctx context.Context, persistentVolumeClaim *v1.PersistentVolumeClaim, opts metav1.CreateOptions) (*v1.PersistentVolumeClaim, error) {
	if persistentVolumeClaim.ObjectMeta.Name == "create_bad" {
		return nil, fmt.Errorf("mock error: cannot create pvc")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete_bad" {
		return fmt.Errorf("mock error: cannot delete pvc")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.PersistentVolumeClaimInterface
func (*PvcMock) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.PersistentVolumeClaim, error) {
	panic("unimplemented")
}

// List implements v1.PersistentVolumeClaimInterface
func (*PvcMock) List(ctx context.Context, opts metav1.ListOptions) (*v1.PersistentVolumeClaimList, error) {
	panic("unimplemented")
}

// Patch implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.PersistentVolumeClaim, err error) {
	panic("unimplemented")
}

// Update implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Update(ctx context.Context, persistentVolumeClaim *v1.PersistentVolumeClaim, opts metav1.UpdateOptions) (*v1.PersistentVolumeClaim, error) {
	if persistentVolumeClaim.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error: cannot update pvc")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, persistentVolumeClaim.ObjectMeta.Name)
	}
}

// UpdateStatus implements v1.PersistentVolumeClaimInterface
func (*PvcMock) UpdateStatus(ctx context.Context, persistentVolumeClaim *v1.PersistentVolumeClaim, opts metav1.UpdateOptions) (*v1.PersistentVolumeClaim, error) {
	panic("unimplemented")
}

// Watch implements v1.PersistentVolumeClaimInterface
func (*PvcMock) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
