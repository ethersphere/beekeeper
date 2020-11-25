package containers

import v1 "k8s.io/api/core/v1"

// EphemeralContainers represents Kubernetes EphemeralContainers
type EphemeralContainers []EphemeralContainer

// ToK8S converts EphemeralContainers to Kuberntes client objects
func (ecs EphemeralContainers) ToK8S() (l []v1.EphemeralContainer) {
	l = make([]v1.EphemeralContainer, 0, len(ecs))

	for _, e := range ecs {
		l = append(l, e.ToK8S())
	}

	return
}

// EphemeralContainer represents Kubernetes EphemeralContainer
type EphemeralContainer struct {
	EphemeralContainerCommon
	TargetContainerName string
}

// ToK8S converts EphemeralContainer to Kuberntes client object
func (ec *EphemeralContainer) ToK8S() v1.EphemeralContainer {
	return v1.EphemeralContainer{
		EphemeralContainerCommon: ec.EphemeralContainerCommon.toK8S(),
		TargetContainerName:      ec.TargetContainerName,
	}
}

// EphemeralContainerCommon represents Kubernetes EphemeralContainerCommon
type EphemeralContainerCommon struct {
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

// ToK8S converts EphemeralContainerCommon to Kuberntes client object
func (ecc *EphemeralContainerCommon) toK8S() v1.EphemeralContainerCommon {
	return v1.EphemeralContainerCommon{
		Name:                     ecc.Name,
		Args:                     ecc.Args,
		Command:                  ecc.Command,
		Env:                      ecc.Env.toK8S(),
		EnvFrom:                  ecc.EnvFrom.toK8S(),
		Image:                    ecc.Image,
		ImagePullPolicy:          v1.PullPolicy(ecc.ImagePullPolicy),
		Lifecycle:                ecc.Lifecycle.toK8S(),
		LivenessProbe:            ecc.LivenessProbe.toK8S(),
		Ports:                    ecc.Ports.toK8S(),
		ReadinessProbe:           ecc.ReadinessProbe.toK8S(),
		Resources:                ecc.Resources.toK8S(),
		SecurityContext:          ecc.SecurityContext.toK8S(),
		StartupProbe:             ecc.StartupProbe.toK8S(),
		Stdin:                    ecc.Stdin,
		StdinOnce:                ecc.StdinOnce,
		TerminationMessagePath:   ecc.TerminationMessagePath,
		TerminationMessagePolicy: v1.TerminationMessagePolicy(ecc.TerminationMessagePolicy),
		TTY:                      ecc.TTY,
		VolumeDevices:            ecc.VolumeDevices.toK8S(),
		VolumeMounts:             ecc.VolumeMounts.toK8S(),
		WorkingDir:               ecc.WorkingDir,
	}
}
