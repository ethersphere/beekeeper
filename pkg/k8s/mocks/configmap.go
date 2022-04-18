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
	cofnigcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.ConfigMapInterface = (*ConfigMapMock)(nil)

type ConfigMapMock struct{}

// Apply implements v1.ConfigMapInterface
func (*ConfigMapMock) Apply(ctx context.Context, configMap *cofnigcorev1.ConfigMapApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ConfigMap, err error) {
	panic("unimplemented")
}

// Create implements v1.ConfigMapInterface
func (*ConfigMapMock) Create(ctx context.Context, configMap *v1.ConfigMap, opts metav1.CreateOptions) (*v1.ConfigMap, error) {
	if configMap.ObjectMeta.Name != "create" {
		return nil, fmt.Errorf("mock error: cannot create config map")
	}
	return &v1.ConfigMap{}, nil
}

// Delete implements v1.ConfigMapInterface
func (*ConfigMapMock) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete" {
		return nil
	} else if name == "delete_not_found" {
		return errors.NewNotFound(schema.GroupResource{}, name)
	} else {
		return fmt.Errorf("mock error: cannot delete config map")
	}
}

// DeleteCollection implements v1.ConfigMapInterface
func (*ConfigMapMock) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.ConfigMapInterface
func (*ConfigMapMock) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ConfigMap, error) {
	panic("unimplemented")
}

// List implements v1.ConfigMapInterface
func (*ConfigMapMock) List(ctx context.Context, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
	panic("unimplemented")
}

// Patch implements v1.ConfigMapInterface
func (*ConfigMapMock) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ConfigMap, err error) {
	panic("unimplemented")
}

// Update implements v1.ConfigMapInterface
func (*ConfigMapMock) Update(ctx context.Context, configMap *v1.ConfigMap, opts metav1.UpdateOptions) (*v1.ConfigMap, error) {
	if configMap.ObjectMeta.Name == "update" {
		return &v1.ConfigMap{}, nil
	} else if configMap.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, configMap.ObjectMeta.Name)
	}
}

// Watch implements v1.ConfigMapInterface
func (*ConfigMapMock) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
