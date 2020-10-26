package containers

import v1 "k8s.io/api/core/v1"

// Lifecycle represents Kubernetes Lifecycle
type Lifecycle struct {
	PostStart *Handler
	PreStop   *Handler
}

// toK8S converts Lifecycle to Kuberntes client object
func (l Lifecycle) toK8S() *v1.Lifecycle {
	if l.PostStart != nil {
		postStart := l.PostStart.toK8S()
		return &v1.Lifecycle{PostStart: &postStart}
	} else if l.PreStop != nil {
		preStop := l.PreStop.toK8S()
		return &v1.Lifecycle{PreStop: &preStop}
	} else {
		return nil
	}
}
