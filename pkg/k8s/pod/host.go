package pod

import v1 "k8s.io/api/core/v1"

// HostAliases represents Kubernetes HostAliases
type HostAliases []HostAlias

// toK8S converts HostAliases to Kubernetes client objects
func (has HostAliases) toK8S() (l []v1.HostAlias) {
	if len(has) > 0 {
		l = make([]v1.HostAlias, 0, len(has))
		for _, h := range has {
			l = append(l, h.toK8S())
		}
	}
	return l
}

// HostAlias represents Kubernetes HostAliase
type HostAlias struct {
	IP        string
	Hostnames []string
}

// toK8S converts HostAliase to Kubernetes client object
func (ha *HostAlias) toK8S() v1.HostAlias {
	return v1.HostAlias{
		IP:        ha.IP,
		Hostnames: ha.Hostnames,
	}
}
