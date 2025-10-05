package pod

import v1 "k8s.io/api/core/v1"

// PodDNSConfig represents Kubernetes Volume
type PodDNSConfig struct {
	Nameservers []string
	Searches    []string
	Options     PodDNSConfigOptions
}

// toK8S converts PodDNSConfig to Kubernetes client object
func (pdc *PodDNSConfig) toK8S() *v1.PodDNSConfig {
	return &v1.PodDNSConfig{
		Nameservers: pdc.Nameservers,
		Searches:    pdc.Searches,
		Options:     pdc.Options.toK8S(),
	}
}

// PodDNSConfigOptions represents Kubernetes PodDNSConfigOptions
type PodDNSConfigOptions []PodDNSConfigOption

// toK8S converts Items to Kubernetes client object
func (pdcos PodDNSConfigOptions) toK8S() (l []v1.PodDNSConfigOption) {
	if len(pdcos) > 0 {
		l = make([]v1.PodDNSConfigOption, 0, len(pdcos))
		for _, p := range pdcos {
			l = append(l, p.toK8S())
		}
	}
	return l
}

// PodDNSConfigOption represents Kubernetes PodDNSConfigOption
type PodDNSConfigOption struct {
	Name  string
	Value string
}

// toK8S converts PodDNSConfigOption to Kubernetes client object
func (pdco *PodDNSConfigOption) toK8S() v1.PodDNSConfigOption {
	return v1.PodDNSConfigOption{
		Name:  pdco.Name,
		Value: &pdco.Value,
	}
}
