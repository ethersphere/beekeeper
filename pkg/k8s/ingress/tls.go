package ingress

import ev1b1 "k8s.io/api/extensions/v1beta1"

// TLSs represents service's IngressTLSs
type TLSs []TLS

// toK8S converts TLSs to Kuberntes client objects
func (ts TLSs) toK8S() (l []ev1b1.IngressTLS) {
	l = make([]ev1b1.IngressTLS, 0, len(ts))

	for _, t := range ts {
		l = append(l, t.toK8S())
	}

	return
}

// TLS represents service's IngressTLS
type TLS struct {
	Hosts      []string
	SecretName string
}

// toK8S converts TLS to Kuberntes client object
func (t TLS) toK8S() (tls ev1b1.IngressTLS) {
	return ev1b1.IngressTLS{
		Hosts: func() (hosts []string) {
			hosts = append(hosts, t.Hosts...)
			return
		}(),
		SecretName: t.SecretName,
	}
}
