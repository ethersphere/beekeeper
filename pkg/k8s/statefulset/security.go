package statefulset

import v1 "k8s.io/api/core/v1"

// SecurityContext ...
type SecurityContext struct {
	AllowPrivilegeEscalation bool
	Capabilities             Capabilities
	Privileged               bool
	ProcMount                string
	ReadOnlyRootFilesystem   bool
	RunAsGroup               int64
	RunAsNonRoot             bool
	RunAsUser                int64
	SELinuxOptions           SELinuxOptions
	WindowsOptions           WindowsOptions
}

func (sc SecurityContext) toK8S() *v1.SecurityContext {
	return &v1.SecurityContext{
		AllowPrivilegeEscalation: &sc.AllowPrivilegeEscalation,
		Capabilities:             sc.Capabilities.toK8S(),
		Privileged:               &sc.Privileged,
		ProcMount: func() *v1.ProcMountType {
			p := v1.ProcMountType(sc.ProcMount)
			return &p
		}(),
		ReadOnlyRootFilesystem: &sc.ReadOnlyRootFilesystem,
		RunAsGroup:             &sc.RunAsGroup,
		RunAsNonRoot:           &sc.RunAsNonRoot,
		RunAsUser:              &sc.RunAsUser,
		SELinuxOptions:         sc.SELinuxOptions.toK8S(),
		// WindowsOptions:         sc.WindowsOptions.toK8S(),
	}
}

// Capabilities ...
type Capabilities struct {
	Add  []string
	Drop []string
}

func (cap Capabilities) toK8S() *v1.Capabilities {
	caps := v1.Capabilities{}
	for _, a := range cap.Add {
		caps.Add = append(caps.Add, v1.Capability(a))
	}
	for _, d := range cap.Drop {
		caps.Add = append(caps.Drop, v1.Capability(d))
	}
	return &caps
}

// SELinuxOptions ...
type SELinuxOptions struct {
	User  string
	Role  string
	Type  string
	Level string
}

func (se SELinuxOptions) toK8S() *v1.SELinuxOptions {
	return &v1.SELinuxOptions{
		User:  se.User,
		Role:  se.Role,
		Type:  se.Type,
		Level: se.Level,
	}
}

// WindowsOptions ...
type WindowsOptions struct {
	GMSACredentialSpecName string
	GMSACredentialSpec     string
	RunAsUserName          string
}

func (ws WindowsOptions) toK8S() *v1.WindowsSecurityContextOptions {
	return &v1.WindowsSecurityContextOptions{
		GMSACredentialSpecName: &ws.GMSACredentialSpecName,
		GMSACredentialSpec:     &ws.GMSACredentialSpec,
		RunAsUserName:          &ws.RunAsUserName,
	}
}
