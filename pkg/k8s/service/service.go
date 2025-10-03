package service

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Spec represents Kubernetes ServiceSpec
type Spec struct {
	ClusterIP                     string
	ExternalIPs                   []string
	ExternalName                  string
	ExternalTrafficPolicy         string
	LoadBalancerIP                string
	LoadBalancerSourceRanges      []string
	Ports                         Ports
	PublishNotReadyAddresses      bool
	Selector                      map[string]string
	SessionAffinity               string
	SessionAffinityTimeoutSeconds int32
	Type                          string
}

// ToK8S converts ServiceSpec to Kubernetes client object
func (s *Spec) ToK8S() v1.ServiceSpec {
	return v1.ServiceSpec{
		ClusterIP:                s.ClusterIP,
		ExternalIPs:              s.ExternalIPs,
		ExternalName:             s.ExternalName,
		ExternalTrafficPolicy:    v1.ServiceExternalTrafficPolicyType(s.ExternalTrafficPolicy),
		LoadBalancerIP:           s.LoadBalancerIP,
		LoadBalancerSourceRanges: s.LoadBalancerSourceRanges,
		Ports:                    s.Ports.toK8S(),
		PublishNotReadyAddresses: s.PublishNotReadyAddresses,
		Selector:                 s.Selector,
		SessionAffinity:          v1.ServiceAffinity(s.SessionAffinity),
		SessionAffinityConfig: &v1.SessionAffinityConfig{
			ClientIP: &v1.ClientIPConfig{
				TimeoutSeconds: &s.SessionAffinityTimeoutSeconds,
			},
		},
		Type: v1.ServiceType(s.Type),
	}
}

// Ports represents service's ports
type Ports []Port

// toK8S converts Ports to Kubernetes client objects
func (ps Ports) toK8S() (l []v1.ServicePort) {
	l = make([]v1.ServicePort, 0, len(ps))

	for _, p := range ps {
		l = append(l, p.toK8S())
	}

	return l
}

// Port represents service's port
type Port struct {
	Name        string
	AppProtocol string
	Nodeport    int32
	Port        int32
	Protocol    string
	TargetPort  string
}

// toK8S converts Port to Kubernetes client object
func (p *Port) toK8S() v1.ServicePort {
	return v1.ServicePort{
		Name:        p.Name,
		AppProtocol: &p.AppProtocol,
		NodePort:    p.Nodeport,
		Port:        p.Port,
		Protocol:    v1.Protocol(p.Protocol),
		TargetPort:  intstr.FromString(p.TargetPort),
	}
}
