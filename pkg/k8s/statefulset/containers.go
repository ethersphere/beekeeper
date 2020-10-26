package statefulset

import (
	v1 "k8s.io/api/core/v1"
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

func containersToK8S(containers []Container) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}

// Lifecycle ...
type Lifecycle struct {
	PostStart *Handler
	PreStop   *Handler
}

func (l Lifecycle) toK8S() *v1.Lifecycle {
	if l.PostStart != nil {
		postStart := l.PostStart.toK8S()
		return &v1.Lifecycle{PostStart: &postStart}
	} else if l.PreStop != nil {
		preStop := l.PreStop.toK8S()
		return &v1.Lifecycle{PreStop: &preStop}
	} else {
		return nil
	}
}

// Port represents containers's port
type Port struct {
	Name          string
	ContainerPort int32
	HostIP        string
	HostPort      int32
	Protocol      string
}

func (p Port) toK8S() v1.ContainerPort {
	return v1.ContainerPort{
		Name:          p.Name,
		ContainerPort: p.ContainerPort,
		HostIP:        p.HostIP,
		HostPort:      p.HostPort,
		Protocol:      v1.Protocol(p.Protocol),
	}
}

func portsToK8S(ports []Port) (ps []v1.ContainerPort) {
	for _, port := range ports {
		ps = append(ps, port.toK8S())
	}
	return
}

// VolumeDevice ...
type VolumeDevice struct {
	Name       string
	DevicePath string
}

func (vd VolumeDevice) toK8S() v1.VolumeDevice {
	return v1.VolumeDevice{
		Name:       vd.Name,
		DevicePath: vd.DevicePath,
	}
}

func volumeDevicesToK8S(volumeDevices []VolumeDevice) (l []v1.VolumeDevice) {
	for _, volumeDevice := range volumeDevices {
		l = append(l, volumeDevice.toK8S())
	}
	return
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
