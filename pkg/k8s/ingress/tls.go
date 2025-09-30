package ingress

import v1 "k8s.io/api/networking/v1"

// TLSs represents service's IngressTLSs
type TLSs []TLS

// toK8S converts TLSs to Kubernetes client objects
func (ts TLSs) toK8S() (l []v1.IngressTLS) {
	if len(ts) > 0 {
		l = make([]v1.IngressTLS, 0, len(ts))

		for _, t := range ts {
			l = append(l, t.toK8S())
		}
	}
	return l
}

// TLS represents service's IngressTLS
type TLS struct {
	Hosts      []string
	SecretName string
}

// toK8S converts TLS to Kubernetes client object
func (t *TLS) toK8S() (tls v1.IngressTLS) {
	return v1.IngressTLS{
		Hosts:      t.Hosts,
		SecretName: t.SecretName,
	}
}
