package mocks

import (
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rest "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.CoreV1Interface = (*CoreV1)(nil)

type CoreV1 struct{}

func NewCoreV1() *CoreV1 {
	return &CoreV1{}
}

// ComponentStatuses implements v1.CoreV1Interface
func (*CoreV1) ComponentStatuses() corev1.ComponentStatusInterface {
	panic("unimplemented")
}

// ConfigMaps implements v1.CoreV1Interface
func (*CoreV1) ConfigMaps(namespace string) corev1.ConfigMapInterface {
	return NewConfigMap()
}

// Endpoints implements v1.CoreV1Interface
func (*CoreV1) Endpoints(namespace string) corev1.EndpointsInterface {
	panic("unimplemented")
}

// Events implements v1.CoreV1Interface
func (*CoreV1) Events(namespace string) corev1.EventInterface {
	panic("unimplemented")
}

// LimitRanges implements v1.CoreV1Interface
func (*CoreV1) LimitRanges(namespace string) corev1.LimitRangeInterface {
	panic("unimplemented")
}

// Namespaces implements v1.CoreV1Interface
func (*CoreV1) Namespaces() corev1.NamespaceInterface {
	return NewNamespace()
}

// Nodes implements v1.CoreV1Interface
func (*CoreV1) Nodes() corev1.NodeInterface {
	panic("unimplemented")
}

// PersistentVolumes implements v1.CoreV1Interface
func (*CoreV1) PersistentVolumes() corev1.PersistentVolumeInterface {
	panic("unimplemented")
}

// PersistentVolumeClaims implements v1.CoreV1Interface
func (*CoreV1) PersistentVolumeClaims(namespace string) corev1.PersistentVolumeClaimInterface {
	return NewPvc()
}

// Pods implements v1.CoreV1Interface
func (*CoreV1) Pods(namespace string) corev1.PodInterface {
	return NewPod()
}

// PodTemplates implements v1.CoreV1Interface
func (*CoreV1) PodTemplates(namespace string) corev1.PodTemplateInterface {
	panic("unimplemented")
}

// ReplicationControllers implements v1.CoreV1Interface
func (*CoreV1) ReplicationControllers(namespace string) corev1.ReplicationControllerInterface {
	panic("unimplemented")
}

// ResourceQuotas implements v1.CoreV1Interface
func (*CoreV1) ResourceQuotas(namespace string) corev1.ResourceQuotaInterface {
	panic("unimplemented")
}

// Secrets implements v1.CoreV1Interface
func (*CoreV1) Secrets(namespace string) corev1.SecretInterface {
	return NewSecret()
}

// Services implements v1.CoreV1Interface
func (*CoreV1) Services(namespace string) corev1.ServiceInterface {
	return NewService()
}

// ServiceAccounts implements v1.CoreV1Interface
func (*CoreV1) ServiceAccounts(namespace string) corev1.ServiceAccountInterface {
	return NewServiceAccount()
}

// RESTClient implements v1.CoreV1Interface
func (*CoreV1) RESTClient() rest.Interface {
	panic("unimplemented")
}
