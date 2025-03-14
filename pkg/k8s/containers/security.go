package containers

import v1 "k8s.io/api/core/v1"

// SecurityContext represents Kubernetes SecurityContext
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

// toK8S converts SecurityContext to Kubernetes client object
func (sc *SecurityContext) toK8S() *v1.SecurityContext {
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
		WindowsOptions:         sc.WindowsOptions.toK8S(),
	}
}

// Capabilities represents Kubernetes Capabilities
type Capabilities struct {
	Add  []string
	Drop []string
}

// toK8S converts Capabilities to Kubernetes client object
func (cap *Capabilities) toK8S() *v1.Capabilities {
	if cap.Add == nil && cap.Drop == nil {
		return nil
	}

	caps := v1.Capabilities{}
	for _, a := range cap.Add {
		caps.Add = append(caps.Add, v1.Capability(a))
	}
	for _, d := range cap.Drop {
		caps.Drop = append(caps.Drop, v1.Capability(d))
	}
	return &caps
}

// SELinuxOptions represents Kubernetes SELinuxOptions
type SELinuxOptions struct {
	User  string
	Role  string
	Type  string
	Level string
}

// toK8S converts SELinuxOptions to Kubernetes client object
func (se *SELinuxOptions) toK8S() *v1.SELinuxOptions {
	return &v1.SELinuxOptions{
		User:  se.User,
		Role:  se.Role,
		Type:  se.Type,
		Level: se.Level,
	}
}

// WindowsOptions represents Kubernetes WindowsSecurityContextOptions
type WindowsOptions struct {
	GMSACredentialSpecName string
	GMSACredentialSpec     string
	RunAsUserName          string
}

func (wo *WindowsOptions) toK8S() *v1.WindowsSecurityContextOptions {
	if wo.GMSACredentialSpecName == "" && wo.GMSACredentialSpec == "" && wo.RunAsUserName == "" {
		return nil
	}
	return &v1.WindowsSecurityContextOptions{
		GMSACredentialSpecName: &wo.GMSACredentialSpecName,
		GMSACredentialSpec:     &wo.GMSACredentialSpec,
		RunAsUserName:          &wo.RunAsUserName,
	}
}
