package container

import (
	v1 "k8s.io/api/core/v1"
)

// Container ...
type Container struct {
	Name                     string
	Args                     []string
	Command                  []string
	Env                      []EnvVar
	EnvFrom                  []EnvFrom
	Image                    string
	ImagePullPolicy          string
	Lifecycle                Lifecycle
	LivenessProbe            Probe
	Ports                    []Port
	ReadinessProbe           Probe
	Resources                Resources
	SecurityContext          SecurityContext
	StartupProbe             Probe
	Stdin                    bool
	StdinOnce                bool
	TerminationMessagePath   string
	TerminationMessagePolicy string
	TTY                      bool
	VolumeDevices            []VolumeDevice
	VolumeMounts             []VolumeMount
	WorkingDir               string
}

// ToK8S ...
func ToK8S(containers []Container) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}

func (c Container) toK8S() v1.Container {
	return v1.Container{
		Name:                     c.Name,
		Args:                     c.Args,
		Command:                  c.Command,
		Env:                      envVarsToK8S(c.Env),
		EnvFrom:                  envFromToK8S(c.EnvFrom),
		Image:                    c.Image,
		ImagePullPolicy:          v1.PullPolicy(c.ImagePullPolicy),
		Lifecycle:                c.Lifecycle.toK8S(),
		LivenessProbe:            c.LivenessProbe.toK8S(),
		Ports:                    portsToK8S(c.Ports),
		ReadinessProbe:           c.ReadinessProbe.toK8S(),
		Resources:                c.Resources.toK8S(),
		SecurityContext:          c.SecurityContext.toK8S(),
		StartupProbe:             c.StartupProbe.toK8S(),
		Stdin:                    c.Stdin,
		StdinOnce:                c.StdinOnce,
		TerminationMessagePath:   c.TerminationMessagePath,
		TerminationMessagePolicy: v1.TerminationMessagePolicy(c.TerminationMessagePolicy),
		TTY:                      c.TTY,
		VolumeDevices:            volumeDevicesToK8S(c.VolumeDevices),
		VolumeMounts:             volumeMountsToK8S(c.VolumeMounts),
		WorkingDir:               c.WorkingDir,
	}
}
