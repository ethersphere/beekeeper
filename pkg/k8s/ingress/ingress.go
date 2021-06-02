package ingress

import v1 "k8s.io/api/networking/v1"

// Spec represents Kubernetes IngressSpec
type Spec struct {
	Class   string
	Backend Backend
	TLS     TLSs
	Rules   Rules
}

// toK8S converts IngressSpec to Kuberntes client object
func (s *Spec) toK8S() v1.IngressSpec {
	return v1.IngressSpec{
		IngressClassName: &s.Class,
		Rules:            s.Rules.toK8S(),
		TLS:              s.TLS.toK8S(),
	}
}
