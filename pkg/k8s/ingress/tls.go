package ingress

import v1 "k8s.io/api/networking/v1"

// TLSs represents service's IngressTLSs
type TLSs []TLS

// toK8S converts TLSs to Kuberntes client objects
func (ts TLSs) toK8S() (l []v1.IngressTLS) {
	if len(ts) > 0 {
		l = make([]v1.IngressTLS, 0, len(ts))

		for _, t := range ts {
			l = append(l, t.toK8S())
		}
	}
	return
}

// TLS represents service's IngressTLS
type TLS struct {
	Hosts      []string
	SecretName string
}

// toK8S converts TLS to Kuberntes client object
func (t *TLS) toK8S() (tls v1.IngressTLS) {
	return v1.IngressTLS{
<<<<<<< HEAD
		Hosts:      t.Hosts,
=======
		Hosts: func() (hosts []string) {
			if len(t.Hosts) > 0 {
				hosts = append(hosts, t.Hosts...)
			}
			return
		}(),
>>>>>>> b4fa6307dbb77ede76aa4cd74c9402481398037d
		SecretName: t.SecretName,
	}
}
