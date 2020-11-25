package containers

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// EnvVars represents Kubernetes EnvVars
type EnvVars []EnvVar

// toK8S converts EnvVars to Kuberntes client objects
func (evs EnvVars) toK8S() (l []v1.EnvVar) {
	l = make([]v1.EnvVar, 0, len(evs))

	for _, e := range evs {
		l = append(l, e.toK8S())
	}

	return
}

// EnvVar represents Kubernetes EnvVar
type EnvVar struct {
	Name      string
	Value     string
	ValueFrom ValueFrom
}

// toK8S converts EnvVar to Kuberntes client object
func (ev *EnvVar) toK8S() v1.EnvVar {
	return v1.EnvVar{
		Name:  ev.Name,
		Value: ev.Value,
		ValueFrom: &v1.EnvVarSource{
			FieldRef:         ev.ValueFrom.Field.toK8S(),
			ResourceFieldRef: ev.ValueFrom.ResourceField.toK8S(),
			ConfigMapKeyRef:  ev.ValueFrom.ConfigMap.toK8S(),
			SecretKeyRef:     ev.ValueFrom.Secret.toK8S(),
		},
	}
}

// ValueFrom represents Kubernetes ValueFrom
type ValueFrom struct {
	Field         Field
	ResourceField ResourceField
	ConfigMap     ConfigMapKey
	Secret        SecretKey
}

// Field represents Kubernetes ObjectFieldSelector
type Field struct {
	APIVersion string
	Path       string
}

// toK8S converts Field to Kuberntes client object
func (f *Field) toK8S() *v1.ObjectFieldSelector {
	return &v1.ObjectFieldSelector{
		APIVersion: f.APIVersion,
		FieldPath:  f.Path,
	}
}

// ResourceField represents Kubernetes ResourceField
type ResourceField struct {
	ContainerName string
	Resource      string
	Divisor       string
}

// toK8S converts ResourceField to Kuberntes client object
func (rf *ResourceField) toK8S() *v1.ResourceFieldSelector {
	return &v1.ResourceFieldSelector{
		ContainerName: rf.ContainerName,
		Resource:      rf.ContainerName,
		Divisor:       resource.MustParse(rf.Divisor),
	}
}

// ConfigMapKey represents Kubernetes ConfigMapKey
type ConfigMapKey struct {
	ConfigMapName string
	Key           string
	Optional      bool
}

// toK8S converts ConfigMapKey to Kuberntes client object
func (cmk *ConfigMapKey) toK8S() *v1.ConfigMapKeySelector {
	return &v1.ConfigMapKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: cmk.ConfigMapName},
		Key:                  cmk.Key,
		Optional:             &cmk.Optional,
	}
}

// SecretKey represents Kubernetes SecretKey
type SecretKey struct {
	SecretName string
	Key        string
	Optional   bool
}

// toK8S converts SecretKey to Kuberntes client object
func (sk *SecretKey) toK8S() *v1.SecretKeySelector {
	return &v1.SecretKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: sk.SecretName},
		Key:                  sk.Key,
		Optional:             &sk.Optional,
	}
}

// EnvFroms represents Kubernetes EnvFromSources
type EnvFroms []EnvFrom

func (efs EnvFroms) toK8S() (l []v1.EnvFromSource) {
	l = make([]v1.EnvFromSource, 0, len(efs))

	for _, ef := range efs {
		l = append(l, ef.toK8S())
	}

	return
}

// EnvFrom represents Kubernetes EnvFromSource
type EnvFrom struct {
	Prefix    string
	ConfigMap ConfigMapRef
	Secret    SecretRef
}

// toK8S converts EnvFrom to Kuberntes client object
func (ef *EnvFrom) toK8S() v1.EnvFromSource {
	return v1.EnvFromSource{
		Prefix:       ef.Prefix,
		ConfigMapRef: ef.ConfigMap.toK8S(),
		SecretRef:    ef.Secret.toK8S(),
	}
}

// ConfigMapRef represents Kubernetes ConfigMapRef
type ConfigMapRef struct {
	Name     string
	Optional bool
}

// toK8S converts ConfigMapRef to Kuberntes client object
func (cm *ConfigMapRef) toK8S() *v1.ConfigMapEnvSource {
	return &v1.ConfigMapEnvSource{
		LocalObjectReference: v1.LocalObjectReference{Name: cm.Name},
		Optional:             &cm.Optional,
	}
}

// SecretRef represents Kubernetes SecretRef
type SecretRef struct {
	Name     string
	Optional bool
}

// toK8S converts SecretRef to Kuberntes client object
func (s *SecretRef) toK8S() *v1.SecretEnvSource {
	return &v1.SecretEnvSource{
		LocalObjectReference: v1.LocalObjectReference{Name: s.Name},
		Optional:             &s.Optional,
	}
}
