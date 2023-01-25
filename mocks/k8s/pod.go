package mocks

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policy "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	configcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.PodInterface = (*Pod)(nil)

type Pod struct{}

func NewPod() *Pod {
	return &Pod{}
}

// Bind implements v1.PodInterface
func (*Pod) Bind(ctx context.Context, binding *v1.Binding, opts metav1.CreateOptions) error {
	panic("unimplemented")
}

// Evict implements v1.PodInterface
func (*Pod) Evict(ctx context.Context, eviction *policy.Eviction) error {
	panic("unimplemented")
}

// EvictV1 implements v1.PodInterface
func (*Pod) EvictV1(ctx context.Context, eviction *policyv1.Eviction) error {
	panic("unimplemented")
}

// EvictV1beta1 implements v1.PodInterface
func (*Pod) EvictV1beta1(ctx context.Context, eviction *policy.Eviction) error {
	panic("unimplemented")
}

// GetLogs implements v1.PodInterface
func (*Pod) GetLogs(name string, opts *v1.PodLogOptions) *restclient.Request {
	panic("unimplemented")
}

// ProxyGet implements v1.PodInterface
func (*Pod) ProxyGet(scheme string, name string, port string, path string, params map[string]string) restclient.ResponseWrapper {
	panic("unimplemented")
}

// Apply implements v1.PodInterface
func (*Pod) Apply(ctx context.Context, pod *configcorev1.PodApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Pod, err error) {
	panic("unimplemented")
}

// ApplyStatus implements v1.PodInterface
func (*Pod) ApplyStatus(ctx context.Context, pod *configcorev1.PodApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Pod, err error) {
	panic("unimplemented")
}

// DeleteCollection implements v1.PodInterface
func (*Pod) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("unimplemented")
}

// Get implements v1.PodInterface
func (*Pod) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Pod, error) {
	panic("unimplemented")
}

// List implements v1.PodInterface
func (*Pod) List(ctx context.Context, opts metav1.ListOptions) (*v1.PodList, error) {
	panic("unimplemented")
}

// Patch implements v1.PodInterface
func (*Pod) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Pod, err error) {
	panic("unimplemented")
}

// UpdateEphemeralContainers implements v1.PodInterface
func (*Pod) UpdateEphemeralContainers(ctx context.Context, podName string, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	panic("unimplemented")
}

// UpdateStatus implements v1.PodInterface
func (*Pod) UpdateStatus(ctx context.Context, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	panic("unimplemented")
}

// Watch implements v1.PodInterface
func (*Pod) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("unimplemented")
}

// Create implements v1.PodInterface
func (*Pod) Create(ctx context.Context, pod *v1.Pod, opts metav1.CreateOptions) (*v1.Pod, error) {
	if pod.ObjectMeta.Name == "create_bad" {
		return nil, fmt.Errorf("mock error: cannot create pod")
	} else {
		return nil, fmt.Errorf("mock error: unknown")
	}
}

// Delete implements v1.PodInterface
func (*Pod) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if name == "delete_bad" {
		return fmt.Errorf("mock error: cannot delete pod")
	} else {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}
}

// Update implements v1.PodInterface
func (*Pod) Update(ctx context.Context, pod *v1.Pod, opts metav1.UpdateOptions) (*v1.Pod, error) {
	if pod.ObjectMeta.Name == "update_bad" {
		return nil, errors.NewBadRequest("mock error: cannot update pod")
	} else {
		return nil, errors.NewNotFound(schema.GroupResource{}, pod.ObjectMeta.Name)
	}
}
