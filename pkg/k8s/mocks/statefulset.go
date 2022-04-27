package mocks

import (
	"context"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	configappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ appsv1.StatefulSetInterface = (*StatefulSetMock)(nil)

type StatefulSetMock struct{}

func NewStatefulSetMock() *StatefulSetMock {
	return &StatefulSetMock{}
}

// Apply implements v1.StatefulSetInterface
func (*StatefulSetMock) Apply(ctx context.Context, statefulSet *configappsv1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.StatefulSet, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.StatefulSetInterface
func (*StatefulSetMock) ApplyStatus(ctx context.Context, statefulSet *configappsv1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.StatefulSet, err error) {
	panic("unimplemented")
}

// Create implements v1.StatefulSetInterface
func (*StatefulSetMock) Create(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.CreateOptions) (*v1.StatefulSet, error) {
	if statefulSet.ObjectMeta.Name == "create_bad" {
		return nil, fmt.Errorf("mock error: cannot create statefulset")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.StatefulSetInterface
func (*StatefulSetMock) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete_bad" {
		return fmt.Errorf("mock error: cannot delete statefulset")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.StatefulSetInterface
func (*StatefulSetMock) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.StatefulSetInterface
func (*StatefulSetMock) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.StatefulSet, error) {
	if name == "statefulset_bad" {
		return nil, fmt.Errorf("mock error: bad request")
	}
	return nil, fmt.Errorf("mock error: unknown")
}

// GetScale implements v1.StatefulSetInterface
func (*StatefulSetMock) GetScale(ctx context.Context, statefulSetName string, options metav1.GetOptions) (*autoscalingv1.Scale, error) {
	panic("unimplemented")
}

// List implements v1.StatefulSetInterface
func (*StatefulSetMock) List(ctx context.Context, opts metav1.ListOptions) (*v1.StatefulSetList, error) {
	panic("unimplemented")
}

// Patch implements v1.StatefulSetInterface
func (*StatefulSetMock) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.StatefulSet, err error) {
	panic("unimplemented")
}

// Update implements v1.StatefulSetInterface
func (*StatefulSetMock) Update(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.UpdateOptions) (*v1.StatefulSet, error) {
	if statefulSet.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error: cannot update statefulset")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, statefulSet.ObjectMeta.Name)
	}
}

// UpdateScale implements v1.StatefulSetInterface
func (*StatefulSetMock) UpdateScale(ctx context.Context, statefulSetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error) {
	panic("unimplemented")
}

// UpdateStatus implements v1.StatefulSetInterface
func (*StatefulSetMock) UpdateStatus(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.UpdateOptions) (*v1.StatefulSet, error) {
	panic("unimplemented")
}

// Watch implements v1.StatefulSetInterface
func (*StatefulSetMock) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}
