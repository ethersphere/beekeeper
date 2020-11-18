package pod

import v1 "k8s.io/api/core/v1"

// HostAliases represents Kubernetes HostAliases
type HostAliases []HostAlias

// toK8S converts HostAliases to Kuberntes client objects
func (has *HostAliases) toK8S() (l []v1.HostAlias) {
	l = make([]v1.HostAlias, 0, len(*has))

	for _, h := range *has {
		l = append(l, h.toK8S())
	}

	return
}

// HostAlias represents Kubernetes HostAliase
type HostAlias struct {
	IP        string
	Hostnames []string
}

// toK8S converts HostAliase to Kuberntes client object
func (ha *HostAlias) toK8S() v1.HostAlias {
	return v1.HostAlias{
		IP:        ha.IP,
		Hostnames: ha.Hostnames,
	}
}
