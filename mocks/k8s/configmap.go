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
var _ corev1.ConfigMapInterface = (*ConfigMap)(nil)

type ConfigMap struct{}

func NewConfigMap() *ConfigMap {
	return &ConfigMap{}
}

// Apply implements v1.ConfigMapInterface
func (*ConfigMap) Apply(ctx context.Context, configMap *cofnigcorev1.ConfigMapApplyConfiguration, opts metav1.ApplyOptions) (result *v1.ConfigMap, err error) {
	panic("unimplemented")
}

// Create implements v1.ConfigMapInterface
func (c *ConfigMap) Create(ctx context.Context, configMap *v1.ConfigMap, opts metav1.CreateOptions) (*v1.ConfigMap, error) {
	if configMap.ObjectMeta.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create config map")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.ConfigMapInterface
func (c *ConfigMap) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete config map")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.ConfigMapInterface
func (*ConfigMap) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.ConfigMapInterface
func (*ConfigMap) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ConfigMap, error) {
	panic("unimplemented")
}

// List implements v1.ConfigMapInterface
func (*ConfigMap) List(ctx context.Context, opts metav1.ListOptions) (*v1.ConfigMapList, error) {
	panic("unimplemented")
}

// Patch implements v1.ConfigMapInterface
func (*ConfigMap) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ConfigMap, err error) {
	panic("unimplemented")
}

// Update implements v1.ConfigMapInterface
func (c *ConfigMap) Update(ctx context.Context, configMap *v1.ConfigMap, opts metav1.UpdateOptions) (*v1.ConfigMap, error) {
	if configMap.ObjectMeta.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update config map")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, configMap.ObjectMeta.Name)
	}
}

// Watch implements v1.ConfigMapInterface
func (*ConfigMap) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
