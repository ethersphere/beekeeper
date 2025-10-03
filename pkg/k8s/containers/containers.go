package containers

import (
	v1 "k8s.io/api/core/v1"
)

// Containers represents Kubernetes Containers
type Containers []Container

// ToK8S converts Containers to Kubernetes client objects
func (cs Containers) ToK8S() (l []v1.Container) {
	if len(cs) > 0 {
		l = make([]v1.Container, 0, len(cs))
		for _, c := range cs {
			l = append(l, c.ToK8S())
		}
	}
	return l
}

// Container represents Kubernetes Container
type Container struct {
	Name                     string
	Args                     []string
	Command                  []string
	Env                      EnvVars
	EnvFrom                  EnvFroms
	Image                    string
	ImagePullPolicy          string
	Lifecycle                Lifecycle
	LivenessProbe            Probe
	Ports                    Ports
	ReadinessProbe           Probe
	Resources                Resources
	SecurityContext          SecurityContext
	StartupProbe             Probe
	Stdin                    bool
	StdinOnce                bool
	TerminationMessagePath   string
	TerminationMessagePolicy string
	TTY                      bool
	VolumeDevices            VolumeDevices
	VolumeMounts             VolumeMounts
	WorkingDir               string
}

// ToK8S converts Container to Kubernetes client object
func (c *Container) ToK8S() v1.Container {
	return v1.Container{
		Name:                     c.Name,
		Args:                     c.Args,
		Command:                  c.Command,
		Env:                      c.Env.toK8S(),
		EnvFrom:                  c.EnvFrom.toK8S(),
		Image:                    c.Image,
		ImagePullPolicy:          v1.PullPolicy(c.ImagePullPolicy),
		Lifecycle:                c.Lifecycle.toK8S(),
		LivenessProbe:            c.LivenessProbe.toK8S(),
		Ports:                    c.Ports.toK8S(),
		ReadinessProbe:           c.ReadinessProbe.toK8S(),
		Resources:                c.Resources.toK8S(),
		SecurityContext:          c.SecurityContext.toK8S(),
		StartupProbe:             c.StartupProbe.toK8S(),
		Stdin:                    c.Stdin,
		StdinOnce:                c.StdinOnce,
		TerminationMessagePath:   c.TerminationMessagePath,
		TerminationMessagePolicy: v1.TerminationMessagePolicy(c.TerminationMessagePolicy),
		TTY:                      c.TTY,
		VolumeDevices:            c.VolumeDevices.toK8S(),
		VolumeMounts:             c.VolumeMounts.toK8S(),
		WorkingDir:               c.WorkingDir,
	}
}
