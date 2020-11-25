package ingress

import (
	ev1b1 "k8s.io/api/extensions/v1beta1"
)

// Spec represents Kubernetes IngressSpec
type Spec struct {
	Class   string
	Backend Backend
	TLS     TLSs
	Rules   Rules
}

// toK8S converts IngressSpec to Kuberntes client object
func (s *Spec) toK8S() ev1b1.IngressSpec {
	return ev1b1.IngressSpec{
		Backend: func() *ev1b1.IngressBackend {
			b := s.Backend.toK8S()
			return &b
		}(),
		IngressClassName: &s.Class,
		Rules:            s.Rules.toK8S(),
		TLS:              s.TLS.toK8S(),
	}
}
