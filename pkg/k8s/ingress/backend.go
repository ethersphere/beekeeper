package ingress

import (
	ev1b1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Backend represents Kubernetes IngressBackend
type Backend struct {
	ServiceName string
	ServicePort string
}

// toK8S converts Backend to Kuberntes client object
func (b Backend) toK8S() ev1b1.IngressBackend {
	return ev1b1.IngressBackend{
		ServiceName: b.ServiceName,
		ServicePort: intstr.FromString(b.ServicePort),
	}
}
