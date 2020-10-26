package containers

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// EnvVar ...
type EnvVar struct {
	Name      string
	Value     string
	ValueFrom ValueFrom
}

func (ev EnvVar) toK8S() v1.EnvVar {
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

func envVarsToK8S(envVars []EnvVar) (l []v1.EnvVar) {
	for _, envVar := range envVars {
		l = append(l, envVar.toK8S())
	}
	return
}

// ValueFrom ...
type ValueFrom struct {
	Field         Field
	ResourceField ResourceField
	ConfigMap     ConfigMapKey
	Secret        SecretKey
}

// Field ...
type Field struct {
	APIVersion string
	Path       string
}

func (f Field) toK8S() *v1.ObjectFieldSelector {
	return &v1.ObjectFieldSelector{
		APIVersion: f.APIVersion,
		FieldPath:  f.Path,
	}
}

// ResourceField ...
type ResourceField struct {
	ContainerName string
	Resource      string
	Divisor       string
}

func (rf ResourceField) toK8S() *v1.ResourceFieldSelector {
	return &v1.ResourceFieldSelector{
		ContainerName: rf.ContainerName,
		Resource:      rf.ContainerName,
		Divisor:       resource.MustParse(rf.Divisor),
	}
}

// ConfigMapKey ...
type ConfigMapKey struct {
	ConfigMapName string
	Key           string
	Optional      bool
}

func (cmk ConfigMapKey) toK8S() *v1.ConfigMapKeySelector {
	return &v1.ConfigMapKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: cmk.ConfigMapName},
		Key:                  cmk.Key,
		Optional:             &cmk.Optional,
	}
}

// SecretKey ...
type SecretKey struct {
	SecretName string
	Key        string
	Optional   bool
}

func (sk SecretKey) toK8S() *v1.SecretKeySelector {
	return &v1.SecretKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: sk.SecretName},
		Key:                  sk.Key,
		Optional:             &sk.Optional,
	}
}

// EnvFrom ...
type EnvFrom struct {
	Prefix    string
	ConfigMap ConfigMapRef
	Secret    SecretRef
}

func (ef EnvFrom) toK8S() v1.EnvFromSource {
	return v1.EnvFromSource{
		Prefix:       ef.Prefix,
		ConfigMapRef: ef.ConfigMap.toK8S(),
		SecretRef:    ef.Secret.toK8S(),
	}
}

func envFromToK8S(envFroms []EnvFrom) (l []v1.EnvFromSource) {
	for _, efs := range envFroms {
		l = append(l, efs.toK8S())
	}
	return
}

// ConfigMapRef ...
type ConfigMapRef struct {
	Name     string
	Optional bool
}

func (cm ConfigMapRef) toK8S() *v1.ConfigMapEnvSource {
	return &v1.ConfigMapEnvSource{
		LocalObjectReference: v1.LocalObjectReference{Name: cm.Name},
		Optional:             &cm.Optional,
	}
}

// SecretRef ...
type SecretRef struct {
	Name     string
	Optional bool
}

func (s SecretRef) toK8S() *v1.SecretEnvSource {
	return &v1.SecretEnvSource{
		LocalObjectReference: v1.LocalObjectReference{Name: s.Name},
		Optional:             &s.Optional,
	}
}
