package ingress

import (
	v1 "k8s.io/api/networking/v1"
)

// Backend represents Kubernetes IngressBackend
type Backend struct {
	ServiceName     string
	ServicePortName string
}

// toK8S converts Backend to Kubernetes client object
func (b *Backend) toK8S() v1.IngressBackend {
	return v1.IngressBackend{
		Service: &v1.IngressServiceBackend{
			Name: b.ServiceName,
			Port: v1.ServiceBackendPort{
				Name: b.ServicePortName,
			},
		},
	}
}
