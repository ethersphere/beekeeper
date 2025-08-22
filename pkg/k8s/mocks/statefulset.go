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
	applyconfigurationsautoscalingv1 "k8s.io/client-go/applyconfigurations/autoscaling/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ appsv1.StatefulSetInterface = (*StatefulSet)(nil)

type StatefulSet struct {
	ns string
}

func NewStatefulSet(ns string) *StatefulSet {
	return &StatefulSet{
		ns: ns,
	}
}

// ApplyScale implements v1.StatefulSetInterface
func (*StatefulSet) ApplyScale(ctx context.Context, statefulSetName string, scale *applyconfigurationsautoscalingv1.ScaleApplyConfiguration, opts metav1.ApplyOptions) (*autoscalingv1.Scale, error) {
	panic("unimplemented")
}

// Apply implements v1.StatefulSetInterface
func (*StatefulSet) Apply(ctx context.Context, statefulSet *configappsv1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.StatefulSet, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.StatefulSetInterface
func (*StatefulSet) ApplyStatus(ctx context.Context, statefulSet *configappsv1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions) (result *v1.StatefulSet, err error) {
	panic("unimplemented")
}

// Create implements v1.StatefulSetInterface
func (*StatefulSet) Create(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.CreateOptions) (*v1.StatefulSet, error) {
	if statefulSet.Name == CreateBad {
		return nil, fmt.Errorf("mock error: cannot create statefulset")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.StatefulSetInterface
func (*StatefulSet) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == DeleteBad {
		return fmt.Errorf("mock error: cannot delete statefulset")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// DeleteCollection implements v1.StatefulSetInterface
func (*StatefulSet) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.StatefulSetInterface
func (*StatefulSet) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.StatefulSet, error) {
	if name == "statefulset_bad" {
		return nil, fmt.Errorf("mock error: bad request")
	}
	return nil, fmt.Errorf("mock error: unknown")
}

// GetScale implements v1.StatefulSetInterface
func (*StatefulSet) GetScale(ctx context.Context, statefulSetName string, options metav1.GetOptions) (*autoscalingv1.Scale, error) {
	panic("unimplemented")
}

// List implements v1.StatefulSetInterface
func (ss *StatefulSet) List(ctx context.Context, opts metav1.ListOptions) (*v1.StatefulSetList, error) {
	if ss.ns == "bad_test" {
		return nil, fmt.Errorf("mock error")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, ss.ns)
	}
}

// Patch implements v1.StatefulSetInterface
func (*StatefulSet) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.StatefulSet, err error) {
	panic("unimplemented")
}

// Update implements v1.StatefulSetInterface
func (*StatefulSet) Update(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.UpdateOptions) (*v1.StatefulSet, error) {
	if statefulSet.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update statefulset")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, statefulSet.Name)
	}
}

// UpdateScale implements v1.StatefulSetInterface
func (*StatefulSet) UpdateScale(ctx context.Context, statefulSetName string, scale *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error) {
	panic("unimplemented")
}

// UpdateStatus implements v1.StatefulSetInterface
func (*StatefulSet) UpdateStatus(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.UpdateOptions) (*v1.StatefulSet, error) {
	panic("unimplemented")
}

// Watch implements v1.StatefulSetInterface
func (*StatefulSet) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	switch opts.FieldSelector {
	case "metadata.name=statefulset_bad":
		return nil, fmt.Errorf("mock error: bad request")
	case "metadata.name=test_statefulset":
		watcher := watch.NewFake()
		go func() {
			defer watcher.Stop()
			watcher.Add(&v1.StatefulSet{
				Status: v1.StatefulSetStatus{
					Replicas:      1,
					ReadyReplicas: 1,
				},
			})
		}()
		return watcher, nil
	case "metadata.name=test_statefulset_watcher_stop":
		watcher := watch.NewFake()
		watcher.Stop()
		return watcher, nil
	case "metadata.name=test_statefulset_context_cancel":
		watcher := watch.NewFake()
		return watcher, nil
	default:
		return nil, fmt.Errorf("mock error: unknown")
	}
}
