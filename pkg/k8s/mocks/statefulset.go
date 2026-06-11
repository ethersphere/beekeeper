package mocks

import (
	"context"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	watch "k8s.io/apimachinery/pkg/watch"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ appsv1.StatefulSetInterface = (*StatefulSet)(nil)

type StatefulSet struct {
	appsv1.StatefulSetInterface
	ns string
}

func NewStatefulSet(ns string) *StatefulSet {
	return &StatefulSet{
		ns: ns,
	}
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

// Get implements v1.StatefulSetInterface
func (*StatefulSet) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.StatefulSet, error) {
	if name == "statefulset_bad" {
		return nil, fmt.Errorf("mock error: bad request")
	}
	return nil, fmt.Errorf("mock error: unknown")
}

// List implements v1.StatefulSetInterface
func (ss *StatefulSet) List(ctx context.Context, opts metav1.ListOptions) (*v1.StatefulSetList, error) {
	if ss.ns == "bad_test" {
		return nil, fmt.Errorf("mock error")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, ss.ns)
	}
}

// Update implements v1.StatefulSetInterface
func (*StatefulSet) Update(ctx context.Context, statefulSet *v1.StatefulSet, opts metav1.UpdateOptions) (*v1.StatefulSet, error) {
	if statefulSet.Name == UpdateBad {
		return nil, errors.NewBadRequest("mock error: cannot update statefulset")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, statefulSet.Name)
	}
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
