package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// InitContainer ...
type InitContainer struct {
	Name            string
	Image           string
	ImagePullPolicy string
	Command         []string
	VolumeMounts    []VolumeMount
}

func (c InitContainer) toK8S() v1.Container {
	return v1.Container{
		Name:         c.Name,
		Image:        c.Image,
		Command:      c.Command,
		VolumeMounts: volumeMountsToK8S(c.VolumeMounts),
	}
}

// Container ...
type Container struct {
	Name            string
	Args            []string
	Command         []string
	Env             []EnvVar
	EnvFrom         []EnvFrom
	Image           string
	ImagePullPolicy string
	// Lifecycle       Lifecycle
	LivenessProbe Probe
	// WorkingDir string

	// VolumeDevices []VolumeDevice
	// StartupProbe *Probe

	// TerminationMessagePath string
	// TerminationMessagePolicy TerminationMessagePolicy
	// Stdin bool
	// StdinOnce bool
	// TTY bool
	Ports []Port

	ReadinessProbe  Probe
	Resources       Resources
	SecurityContext SecurityContext
	VolumeMounts    []VolumeMount
}

func (c Container) toK8S() v1.Container {
	return v1.Container{
		Name:            c.Name,
		Args:            c.Args,
		Command:         c.Command,
		Env:             envVarsToK8S(c.Env),
		EnvFrom:         envFromToK8S(c.EnvFrom),
		Image:           c.Image,
		ImagePullPolicy: v1.PullPolicy(c.ImagePullPolicy),
		LivenessProbe: func() *v1.Probe {
			if len(c.LivenessProbe.Path) > 0 && len(c.LivenessProbe.Port) > 0 {
				return &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{
					Path: c.LivenessProbe.Path,
					Port: intstr.FromString(c.LivenessProbe.Port),
				}}}
			}
			return nil
		}(),
		Ports: portsToK8S(c.Ports),
		ReadinessProbe: func() *v1.Probe {
			if len(c.ReadinessProbe.Path) > 0 && len(c.ReadinessProbe.Port) > 0 {
				return &v1.Probe{Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{
					Path: c.ReadinessProbe.Path,
					Port: intstr.FromString(c.ReadinessProbe.Port),
				}}}
			}
			return nil
		}(),
		Resources: v1.ResourceRequirements{
			Limits: func() v1.ResourceList {
				m := map[v1.ResourceName]resource.Quantity{}
				if len(c.Resources.RequestCPU) > 0 {
					m[v1.ResourceCPU] = resource.MustParse(c.Resources.LimitCPU)
				}
				if len(c.Resources.RequestMemory) > 0 {
					m[v1.ResourceMemory] = resource.MustParse(c.Resources.LimitMemory)
				}
				return m
			}(),
			Requests: func() v1.ResourceList {
				m := map[v1.ResourceName]resource.Quantity{}
				if len(c.Resources.RequestCPU) > 0 {
					m[v1.ResourceCPU] = resource.MustParse(c.Resources.RequestCPU)
				}
				if len(c.Resources.RequestMemory) > 0 {
					m[v1.ResourceMemory] = resource.MustParse(c.Resources.RequestMemory)
				}
				return m
			}(),
		},
		SecurityContext: func() *v1.SecurityContext {
			if &c.SecurityContext.AllowPrivilegeEscalation != nil && &c.SecurityContext.RunAsUser != nil {
				return &v1.SecurityContext{
					AllowPrivilegeEscalation: &c.SecurityContext.AllowPrivilegeEscalation,
					RunAsUser:                &c.SecurityContext.RunAsUser,
				}
			}
			return nil
		}(),
		VolumeMounts: volumeMountsToK8S(c.VolumeMounts),
	}
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
