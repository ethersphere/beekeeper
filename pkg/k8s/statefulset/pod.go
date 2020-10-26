package statefulset

import (
	"github.com/ethersphere/beekeeper/pkg/k8s/containers"
	v1 "k8s.io/api/core/v1"
)

// PodSpec represents Kubernetes PodSpec
type PodSpec struct {
	AutomountServiceAccountToken  bool
	Containers                    containers.Containers
	DNSPolicy                     string
	EnableServiceLinks            bool
	Hostname                      string
	HostNetwork                   bool
	HostPID                       bool
	HostIPC                       bool
	InitContainers                containers.Containers
	NodeName                      string
	NodeSelector                  map[string]string
	PodSecurityContext            PodSecurityContext
	Priority                      int32
	PriorityClassName             string
	RestartPolicy                 string
	SchedulerName                 string
	ServiceAccountName            string
	ShareProcessNamespace         bool
	Subdomain                     string
	TerminationGracePeriodSeconds int64
	Volumes                       Volumes
}

func (ps PodSpec) toK8S() v1.PodSpec {
	return v1.PodSpec{
		// TODO: Affinity *Affinity
		AutomountServiceAccountToken: &ps.AutomountServiceAccountToken,
		Containers:                   ps.Containers.ToK8S(),
		// TODO: DNSConfig *PodDNSConfig
		DNSPolicy:          v1.DNSPolicy(ps.DNSPolicy),
		EnableServiceLinks: &ps.EnableServiceLinks,
		// TODO: EphemeralContainers: ps.EphemeralContainers.ToK8S(),
		// TODO: HostAliases []HostAlias
		Hostname:    ps.Hostname,
		HostNetwork: ps.HostNetwork,
		HostPID:     ps.HostPID,
		HostIPC:     ps.HostIPC,
		// TODO: ImagePullSecrets []LocalObjectReference
		InitContainers: ps.InitContainers.ToK8S(),
		NodeName:       ps.NodeName,
		NodeSelector:   ps.NodeSelector,
		// TODO: Overhead ResourceList
		// TODO: PreemptionPolicy *PreemptionPolicy
		Priority:          &ps.Priority,
		PriorityClassName: ps.PriorityClassName,
		// TODO: ReadinessGates []PodReadinessGate
		RestartPolicy:                 v1.RestartPolicy(ps.RestartPolicy),
		SchedulerName:                 ps.SchedulerName,
		SecurityContext:               ps.PodSecurityContext.toK8S(),
		ServiceAccountName:            ps.ServiceAccountName,
		ShareProcessNamespace:         &ps.ShareProcessNamespace,
		Subdomain:                     ps.Subdomain,
		TerminationGracePeriodSeconds: &ps.TerminationGracePeriodSeconds,
		// TODO: Tolerations []Toleration
		// TODO: TopologySpreadConstraints []TopologySpreadConstraint
		Volumes: ps.Volumes.toK8S(),
	}
}
