package statefulset

import (
	"github.com/ethersphere/beekeeper/pkg/k8s/containers"
	v1 "k8s.io/api/core/v1"
)

// PodSpec represents Kubernetes PodSpec
type PodSpec struct {
	Affinity                      Affinity
	AutomountServiceAccountToken  bool
	Containers                    containers.Containers
	DNSConfig                     PodDNSConfig
	DNSPolicy                     string
	EphemeralContainers           containers.EphemeralContainers
	EnableServiceLinks            bool
	HostAliases                   HostAliases
	Hostname                      string
	HostNetwork                   bool
	HostPID                       bool
	HostIPC                       bool
	ImagePullSecrets              []string
	InitContainers                containers.Containers
	NodeName                      string
	NodeSelector                  map[string]string
	PodSecurityContext            PodSecurityContext
	PreemptionPolicy              string
	Priority                      int32
	PriorityClassName             string
	ReadinessGates                PodReadinessGates
	RestartPolicy                 string
	SchedulerName                 string
	ServiceAccountName            string
	ShareProcessNamespace         bool
	Subdomain                     string
	TerminationGracePeriodSeconds int64
	Tolerations                   Tolerations
	TopologySpreadConstraints     TopologySpreadConstraints
	Volumes                       Volumes
}

func (ps PodSpec) toK8S() v1.PodSpec {
	return v1.PodSpec{
		Affinity:                     ps.Affinity.toK8S(),
		AutomountServiceAccountToken: &ps.AutomountServiceAccountToken,
		Containers:                   ps.Containers.ToK8S(),
		DNSConfig:                    ps.DNSConfig.toK8S(),
		DNSPolicy:                    v1.DNSPolicy(ps.DNSPolicy),
		EnableServiceLinks:           &ps.EnableServiceLinks,
		EphemeralContainers:          ps.EphemeralContainers.ToK8S(),
		HostAliases:                  ps.HostAliases.toK8S(),
		Hostname:                     ps.Hostname,
		HostNetwork:                  ps.HostNetwork,
		HostPID:                      ps.HostPID,
		HostIPC:                      ps.HostIPC,
		ImagePullSecrets: func() (l []v1.LocalObjectReference) {
			l = make([]v1.LocalObjectReference, 0, len(ps.ImagePullSecrets))
			for _, i := range ps.ImagePullSecrets {
				l = append(l, v1.LocalObjectReference{Name: i})
			}
			return
		}(),
		InitContainers: ps.InitContainers.ToK8S(),
		NodeName:       ps.NodeName,
		NodeSelector:   ps.NodeSelector,
		PreemptionPolicy: func() *v1.PreemptionPolicy {
			p := v1.PreemptionPolicy(ps.PreemptionPolicy)
			return &p
		}(),
		Priority:                      &ps.Priority,
		PriorityClassName:             ps.PriorityClassName,
		ReadinessGates:                ps.ReadinessGates.toK8S(),
		RestartPolicy:                 v1.RestartPolicy(ps.RestartPolicy),
		SchedulerName:                 ps.SchedulerName,
		SecurityContext:               ps.PodSecurityContext.toK8S(),
		ServiceAccountName:            ps.ServiceAccountName,
		ShareProcessNamespace:         &ps.ShareProcessNamespace,
		Subdomain:                     ps.Subdomain,
		TerminationGracePeriodSeconds: &ps.TerminationGracePeriodSeconds,
		Tolerations:                   ps.Tolerations.toK8S(),
		TopologySpreadConstraints:     ps.TopologySpreadConstraints.toK8S(),
		Volumes:                       ps.Volumes.toK8S(),
	}
}

// PodDNSConfig represents Kubernetes Volume
type PodDNSConfig struct {
	Nameservers []string
	Searches    []string
	Options     PodDNSConfigOptions
}

// toK8S converts PodDNSConfig to Kuberntes client object
func (pdc PodDNSConfig) toK8S() *v1.PodDNSConfig {
	return &v1.PodDNSConfig{
		Nameservers: pdc.Nameservers,
		Searches:    pdc.Searches,
		Options:     pdc.Options.toK8S(),
	}
}

// PodDNSConfigOptions represents Kubernetes PodDNSConfigOptions
type PodDNSConfigOptions []PodDNSConfigOption

// toK8S converts Items to Kuberntes client object
func (pdcos PodDNSConfigOptions) toK8S() (l []v1.PodDNSConfigOption) {
	l = make([]v1.PodDNSConfigOption, 0, len(pdcos))

	for _, p := range pdcos {
		l = append(l, p.toK8S())
	}

	return
}

// PodDNSConfigOption represents Kubernetes PodDNSConfigOption
type PodDNSConfigOption struct {
	Name  string
	Value string
}

// toK8S converts PodDNSConfigOption to Kuberntes client object
func (pdco PodDNSConfigOption) toK8S() v1.PodDNSConfigOption {
	return v1.PodDNSConfigOption{
		Name:  pdco.Name,
		Value: &pdco.Value,
	}
}

// HostAliases represents Kubernetes HostAliases
type HostAliases []HostAlias

// toK8S converts HostAliases to Kuberntes client objects
func (has HostAliases) toK8S() (l []v1.HostAlias) {
	l = make([]v1.HostAlias, 0, len(has))

	for _, h := range has {
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
func (ha HostAlias) toK8S() v1.HostAlias {
	return v1.HostAlias{
		IP:        ha.IP,
		Hostnames: ha.Hostnames,
	}
}

// PodReadinessGates represents Kubernetes PodReadinessGates
type PodReadinessGates []PodReadinessGate

// toK8S converts PodReadinessGates to Kuberntes client objects
func (prgs PodReadinessGates) toK8S() (l []v1.PodReadinessGate) {
	l = make([]v1.PodReadinessGate, 0, len(prgs))

	for _, g := range prgs {
		l = append(l, g.toK8S())
	}

	return
}

// PodReadinessGate represents Kubernetes PodReadinessGate
type PodReadinessGate struct {
	ConditionType string
}

// toK8S converts PodReadinessGate to Kuberntes client object
func (prg PodReadinessGate) toK8S() v1.PodReadinessGate {
	return v1.PodReadinessGate{
		ConditionType: v1.PodConditionType(prg.ConditionType),
	}
}
