package containers

import v1 "k8s.io/api/core/v1"

// Ports represents Kubernetes ContainerPorts
type Ports []Port

// toK8S converts Ports to Kuberntes client object
func (ps Ports) toK8S() (l []v1.ContainerPort) {
	l = make([]v1.ContainerPort, 0, len(ps))

	for _, p := range ps {
		l = append(l, p.toK8S())
	}
	return
}

// Port represents Kubernetes ContainerPort
type Port struct {
	Name          string
	ContainerPort int32
	HostIP        string
	HostPort      int32
	Protocol      string
}

// toK8S converts Port to Kuberntes client object
func (p *Port) toK8S() v1.ContainerPort {
	return v1.ContainerPort{
		Name:          p.Name,
		ContainerPort: p.ContainerPort,
		HostIP:        p.HostIP,
		HostPort:      p.HostPort,
		Protocol:      v1.Protocol(p.Protocol),
	}
}
