package containers

import v1 "k8s.io/api/core/v1"

// Port represents containers's port
type Port struct {
	Name          string
	ContainerPort int32
	HostIP        string
	HostPort      int32
	Protocol      string
}

func (p Port) toK8S() v1.ContainerPort {
	return v1.ContainerPort{
		Name:          p.Name,
		ContainerPort: p.ContainerPort,
		HostIP:        p.HostIP,
		HostPort:      p.HostPort,
		Protocol:      v1.Protocol(p.Protocol),
	}
}

func portsToK8S(ports []Port) (ps []v1.ContainerPort) {
	for _, port := range ports {
		ps = append(ps, port.toK8S())
	}
	return
}
