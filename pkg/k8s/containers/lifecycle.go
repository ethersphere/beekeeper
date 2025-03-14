package containers

import v1 "k8s.io/api/core/v1"

// Lifecycle represents Kubernetes Lifecycle
type Lifecycle struct {
	PostStart *LifecycleHandler
	PreStop   *LifecycleHandler
}

// toK8S converts Lifecycle to Kubernetes client object
func (l *Lifecycle) toK8S() *v1.Lifecycle {
	if l.PostStart == nil && l.PreStop == nil {
		return nil
	}

	var lifecycle v1.Lifecycle

	if l.PostStart != nil {
		postStart := l.PostStart.toK8S()
		lifecycle.PostStart = &postStart
	}

	if l.PreStop != nil {
		preStop := l.PreStop.toK8S()
		lifecycle.PreStop = &preStop
	}

	return &lifecycle
}
