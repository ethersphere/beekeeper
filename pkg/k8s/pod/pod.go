package pod

import (
	"github.com/ethersphere/beekeeper/pkg/k8s/containers"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodTemplateSpec represents Kubernetes PodTemplateSpec
type PodTemplateSpec struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Spec        PodSpec
}

// ToK8S converts PodTemplateSpec to Kubernetes client objects
func (pts *PodTemplateSpec) ToK8S() v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pts.Name,
			Namespace:   pts.Namespace,
			Annotations: pts.Annotations,
			Labels:      pts.Labels,
		},
		Spec: pts.Spec.toK8S(),
	}
}

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

// toK8S converts PodSpec to Kubernetes client object
func (p *PodSpec) toK8S() v1.PodSpec {
	return v1.PodSpec{
		Affinity:                     p.Affinity.toK8S(),
		AutomountServiceAccountToken: &p.AutomountServiceAccountToken,
		Containers:                   p.Containers.ToK8S(),
		DNSConfig:                    p.DNSConfig.toK8S(),
		DNSPolicy:                    v1.DNSPolicy(p.DNSPolicy),
		EnableServiceLinks:           &p.EnableServiceLinks,
		EphemeralContainers:          p.EphemeralContainers.ToK8S(),
		HostAliases:                  p.HostAliases.toK8S(),
		Hostname:                     p.Hostname,
		HostNetwork:                  p.HostNetwork,
		HostPID:                      p.HostPID,
		HostIPC:                      p.HostIPC,
		ImagePullSecrets: func() (l []v1.LocalObjectReference) {
			if len(p.ImagePullSecrets) > 0 {
				l = make([]v1.LocalObjectReference, 0, len(p.ImagePullSecrets))
				for _, i := range p.ImagePullSecrets {
					l = append(l, v1.LocalObjectReference{Name: i})
				}
			}
			return
		}(),
		InitContainers: p.InitContainers.ToK8S(),
		NodeName:       p.NodeName,
		NodeSelector:   p.NodeSelector,
		PreemptionPolicy: func() *v1.PreemptionPolicy {
			if len(p.PreemptionPolicy) > 0 {
				pp := v1.PreemptionPolicy(p.PreemptionPolicy)
				return &pp
			}
			return nil
		}(),
		Priority:                      &p.Priority,
		PriorityClassName:             p.PriorityClassName,
		ReadinessGates:                p.ReadinessGates.toK8S(),
		RestartPolicy:                 v1.RestartPolicy(p.RestartPolicy),
		SchedulerName:                 p.SchedulerName,
		SecurityContext:               p.PodSecurityContext.toK8S(),
		ServiceAccountName:            p.ServiceAccountName,
		ShareProcessNamespace:         &p.ShareProcessNamespace,
		Subdomain:                     p.Subdomain,
		TerminationGracePeriodSeconds: &p.TerminationGracePeriodSeconds,
		Tolerations:                   p.Tolerations.toK8S(),
		TopologySpreadConstraints:     p.TopologySpreadConstraints.toK8S(),
		Volumes:                       p.Volumes.toK8S(),
	}
}
