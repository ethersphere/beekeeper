package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// InitContainer ...
type InitContainer struct {
	Name         string
	Image        string
	Command      []string
	VolumeMounts []VolumeMount
}

func (c InitContainer) toK8S() v1.Container {
	return v1.Container{
		Name:         c.Name,
		Image:        c.Image,
		Command:      c.Command,
		VolumeMounts: volumeMountsToK8S(c.VolumeMounts),
	}
}

func initContainersToK8S(containers []InitContainer) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}

// Container ...
type Container struct {
	Name            string
	Image           string
	ImagePullPolicy string
	Command         []string
	Ports           []Port
	LivenessProbe   Probe
	ReadinessProbe  Probe
	Resources       Resources
	SecurityContext SecurityContext
	VolumeMounts    []VolumeMount
}

func (c Container) toK8S() v1.Container {
	return v1.Container{
		Name:            c.Name,
		Image:           c.Image,
		ImagePullPolicy: v1.PullPolicy(c.ImagePullPolicy),
		Command:         c.Command,
		LivenessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{
			Path: c.LivenessProbe.Path,
			Port: intstr.FromString(c.LivenessProbe.Port),
		}}},
		Ports: portsToK8S(c.Ports),
		ReadinessProbe: &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{
			Path: c.ReadinessProbe.Path,
			Port: intstr.FromString(c.ReadinessProbe.Port),
		}}},
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.Quantity{Format: resource.Format(c.Resources.LimitCPU)},
				v1.ResourceMemory: resource.Quantity{Format: resource.Format(c.Resources.LimitMemory)},
			},
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.Quantity{Format: resource.Format(c.Resources.RequestCPU)},
				v1.ResourceMemory: resource.Quantity{Format: resource.Format(c.Resources.RequestMemory)},
			},
		},
		SecurityContext: &v1.SecurityContext{
			AllowPrivilegeEscalation: &c.SecurityContext.AllowPrivilegeEscalation,
			RunAsUser:                &c.SecurityContext.RunAsUser,
		},
		VolumeMounts: volumeMountsToK8S(c.VolumeMounts),
	}
}

func containersToK8S(containers []Container) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}

// Port represents containers's port
type Port struct {
	Name          string
	Protocol      string
	ContainerPort int32
}

func (p Port) toK8S() v1.ContainerPort {
	return v1.ContainerPort{
		Name:          p.Name,
		Protocol:      v1.Protocol(p.Protocol),
		ContainerPort: p.ContainerPort,
	}

}

func portsToK8S(ports []Port) (ps []v1.ContainerPort) {
	for _, port := range ports {
		ps = append(ps, port.toK8S())
	}
	return
}

// Probe ...
type Probe struct {
	Path string
	Port string
}

// Resources ...
type Resources struct {
	LimitCPU      string
	LimitMemory   string
	RequestCPU    string
	RequestMemory string
}

// SecurityContext ...
type SecurityContext struct {
	AllowPrivilegeEscalation bool
	RunAsUser                int64
}

// VolumeMount ...
type VolumeMount struct {
	Name      string
	MountPath string
	SubPath   string
	ReadOnly  bool
}

func (v VolumeMount) toK8S() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      v.Name,
		MountPath: v.MountPath,
		SubPath:   v.SubPath,
		ReadOnly:  v.ReadOnly,
	}
}

func volumeMountsToK8S(volumeMounts []VolumeMount) (vms []v1.VolumeMount) {
	for _, volumeMount := range volumeMounts {
		vms = append(vms, volumeMount.toK8S())
	}
	return
}
