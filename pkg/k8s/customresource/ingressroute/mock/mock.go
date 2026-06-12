// Package mock provides a hand-written mock of ingressroute.Interface built
// with the functional-options ("With") pattern. IngressRoute is a
// beekeeper-owned custom resource with no upstream fake, so unlike the rest of
// pkg/k8s (which uses client-go's fake clientset + reactors) it needs a bespoke
// double. The mock is safe for concurrent use.
package mock

import (
	"context"
	"sync"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

// Compile-time interface compliance.
var (
	_ ingressroute.Interface             = (*Clientset)(nil)
	_ ingressroute.IngressRouteInterface = (*ingressRoutes)(nil)
)

// Clientset is a mock implementation of ingressroute.Interface.
type Clientset struct {
	mu      sync.Mutex
	objects map[string]ingressroute.IngressRoute // keyed by "namespace/name"

	getErr    error
	createErr error
	updateErr error
	listErr   error
	deleteErr error
}

// Option configures a Clientset.
type Option func(*Clientset)

// New builds a Clientset with the given options.
func New(opts ...Option) *Clientset {
	cs := &Clientset{objects: make(map[string]ingressroute.IngressRoute)}
	for _, opt := range opts {
		opt(cs)
	}
	return cs
}

// WithIngressRoutes seeds the clientset with existing IngressRoutes.
func WithIngressRoutes(irs ...ingressroute.IngressRoute) Option {
	return func(cs *Clientset) {
		for _, ir := range irs {
			cs.objects[key(ir.Namespace, ir.Name)] = ir
		}
	}
}

// WithGetError makes Get fail with err.
func WithGetError(err error) Option { return func(cs *Clientset) { cs.getErr = err } }

// WithCreateError makes Create fail with err.
func WithCreateError(err error) Option { return func(cs *Clientset) { cs.createErr = err } }

// WithUpdateError makes Update fail with err.
func WithUpdateError(err error) Option { return func(cs *Clientset) { cs.updateErr = err } }

// WithListError makes List fail with err.
func WithListError(err error) Option { return func(cs *Clientset) { cs.listErr = err } }

// WithDeleteError makes Delete fail with err.
func WithDeleteError(err error) Option { return func(cs *Clientset) { cs.deleteErr = err } }

func key(namespace, name string) string { return namespace + "/" + name }

// IngressRoutes returns a namespaced IngressRouteInterface.
func (cs *Clientset) IngressRoutes(namespace string) ingressroute.IngressRouteInterface {
	return &ingressRoutes{cs: cs, ns: namespace}
}

// ingressRoutes implements ingressroute.IngressRouteInterface for one namespace.
type ingressRoutes struct {
	cs *Clientset
	ns string
}

func (i *ingressRoutes) Get(_ context.Context, name string, _ metav1.GetOptions) (*ingressroute.IngressRoute, error) {
	i.cs.mu.Lock()
	defer i.cs.mu.Unlock()
	if i.cs.getErr != nil {
		return nil, i.cs.getErr
	}
	ir, ok := i.cs.objects[key(i.ns, name)]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Resource: ingressroute.IngressRouteResource}, name)
	}
	return &ir, nil
}

func (i *ingressRoutes) List(_ context.Context, _ metav1.ListOptions) (*ingressroute.IngressRouteList, error) {
	i.cs.mu.Lock()
	defer i.cs.mu.Unlock()
	if i.cs.listErr != nil {
		return nil, i.cs.listErr
	}
	// The label selector is ignored; tests seed only the objects they expect.
	list := &ingressroute.IngressRouteList{}
	for _, ir := range i.cs.objects {
		if ir.Namespace == i.ns {
			list.Items = append(list.Items, ir)
		}
	}
	return list, nil
}

func (i *ingressRoutes) Create(_ context.Context, ir *ingressroute.IngressRoute) (*ingressroute.IngressRoute, error) {
	i.cs.mu.Lock()
	defer i.cs.mu.Unlock()
	if i.cs.createErr != nil {
		return nil, i.cs.createErr
	}
	i.cs.objects[key(i.ns, ir.Name)] = *ir
	return ir, nil
}

func (i *ingressRoutes) Update(_ context.Context, ir *ingressroute.IngressRoute, _ metav1.UpdateOptions) (*ingressroute.IngressRoute, error) {
	i.cs.mu.Lock()
	defer i.cs.mu.Unlock()
	if i.cs.updateErr != nil {
		return nil, i.cs.updateErr
	}
	i.cs.objects[key(i.ns, ir.Name)] = *ir
	return ir, nil
}

func (i *ingressRoutes) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	i.cs.mu.Lock()
	defer i.cs.mu.Unlock()
	if i.cs.deleteErr != nil {
		return i.cs.deleteErr
	}
	k := key(i.ns, name)
	if _, ok := i.cs.objects[k]; !ok {
		return apierrors.NewNotFound(schema.GroupResource{Resource: ingressroute.IngressRouteResource}, name)
	}
	delete(i.cs.objects, k)
	return nil
}

// Watch is required by the interface but unused by ingressroute.Client; it
// returns an empty watcher.
func (i *ingressRoutes) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return watch.NewRaceFreeFake(), nil
}
