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
var _ corev1.ConfigMapInterface = (*ConfigMap)(nil)

type ConfigMap struct {
	corev1.ConfigMapInterface
}

func NewConfigMap() *ConfigMap {
	return &ConfigMap{}
}

// Create implements v1.ConfigMapInterface
func (c *ConfigMap) Create(ctx context.Context, configMap *v1.ConfigMap, opts metav1.CreateOptions) (*v1.ConfigMap, error) {
	if configMap.Name == CreateBad {
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

// Update implements v1.ConfigMapInterface
func (c *ConfigMap) Update(ctx context.Context, configMap *v1.ConfigMap, opts metav1.UpdateOptions) (*v1.ConfigMap, error) {
	if configMap.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update config map")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, configMap.Name)
	}
}
