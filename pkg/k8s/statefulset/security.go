package statefulset

import (
	v1 "k8s.io/api/core/v1"
)

// PodSecurityContext represents Kubernetes PodSecurityContext
type PodSecurityContext struct {
	FSGroup             int64
	FSGroupChangePolicy string
	RunAsGroup          int64
	RunAsNonRoot        bool
	RunAsUser           int64
	SELinuxOptions      SELinuxOptions
	SupplementalGroups  []int64
	Sysctls             Sysctls
	WindowsOptions      WindowsOptions
}

// toK8S converts PodSecurityContext to Kuberntes client object
func (psc PodSecurityContext) toK8S() *v1.PodSecurityContext {
	return &v1.PodSecurityContext{
		FSGroup: &psc.FSGroup,
		FSGroupChangePolicy: func() *v1.PodFSGroupChangePolicy {
			f := v1.PodFSGroupChangePolicy(psc.FSGroupChangePolicy)
			return &f
		}(),
		RunAsGroup:         &psc.RunAsGroup,
		RunAsNonRoot:       &psc.RunAsNonRoot,
		RunAsUser:          &psc.RunAsUser,
		SELinuxOptions:     psc.SELinuxOptions.toK8S(),
		SupplementalGroups: psc.SupplementalGroups,
		Sysctls:            psc.Sysctls.toK8S(),
	}
}

// SELinuxOptions represents Kubernetes SELinuxOptions
type SELinuxOptions struct {
	User  string
	Role  string
	Type  string
	Level string
}

// toK8S converts SELinuxOptions to Kuberntes client object
func (se SELinuxOptions) toK8S() *v1.SELinuxOptions {
	return &v1.SELinuxOptions{
		User:  se.User,
		Role:  se.Role,
		Type:  se.Type,
		Level: se.Level,
	}
}

// Sysctls represents Kubernetes Sysctls
type Sysctls []Sysctl

// toK8S converts Sysctls to Kuberntes client objects
func (scs Sysctls) toK8S() (l []v1.Sysctl) {
	l = make([]v1.Sysctl, 0, len(scs))

	for _, s := range scs {
		l = append(l, s.toK8S())
	}

	return
}

// Sysctl represents Kubernetes Sysctl
type Sysctl struct {
	Name  string
	Value string
}

func (sc Sysctl) toK8S() v1.Sysctl {
	return v1.Sysctl{
		Name:  sc.Name,
		Value: sc.Value,
	}
}

// WindowsOptions represents Kubernetes WindowsSecurityContextOptions
type WindowsOptions struct {
	GMSACredentialSpecName string
	GMSACredentialSpec     string
	RunAsUserName          string
}

// toK8S converts WindowsOptions to Kuberntes client object
func (ws WindowsOptions) toK8S() *v1.WindowsSecurityContextOptions {
	return &v1.WindowsSecurityContextOptions{
		GMSACredentialSpecName: &ws.GMSACredentialSpecName,
		GMSACredentialSpec:     &ws.GMSACredentialSpec,
		RunAsUserName:          &ws.RunAsUserName,
	}
}
