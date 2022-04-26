package mocks

import (
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rest "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.CoreV1Interface = (*CoreV1Mock)(nil)

type CoreV1Mock struct{}

func NewCoreV1Mock() *CoreV1Mock {
	return &CoreV1Mock{}
}

// ComponentStatuses implements v1.CoreV1Interface
func (*CoreV1Mock) ComponentStatuses() corev1.ComponentStatusInterface {
	panic("unimplemented")
}

// ConfigMaps implements v1.CoreV1Interface
func (*CoreV1Mock) ConfigMaps(namespace string) corev1.ConfigMapInterface {
	return NewConfigMapMock()
}

// Endpoints implements v1.CoreV1Interface
func (*CoreV1Mock) Endpoints(namespace string) corev1.EndpointsInterface {
	panic("unimplemented")
}

// Events implements v1.CoreV1Interface
func (*CoreV1Mock) Events(namespace string) corev1.EventInterface {
	panic("unimplemented")
}

// LimitRanges implements v1.CoreV1Interface
func (*CoreV1Mock) LimitRanges(namespace string) corev1.LimitRangeInterface {
	panic("unimplemented")
}

// Namespaces implements v1.CoreV1Interface
func (*CoreV1Mock) Namespaces() corev1.NamespaceInterface {
	return NewNamespaceMock()
}

// Nodes implements v1.CoreV1Interface
func (*CoreV1Mock) Nodes() corev1.NodeInterface {
	panic("unimplemented")
}

// PersistentVolumes implements v1.CoreV1Interface
func (*CoreV1Mock) PersistentVolumes() corev1.PersistentVolumeInterface {
	panic("unimplemented")
}

// PersistentVolumeClaims implements v1.CoreV1Interface
func (*CoreV1Mock) PersistentVolumeClaims(namespace string) corev1.PersistentVolumeClaimInterface {
	return NewPvcMock()
}

// Pods implements v1.CoreV1Interface
func (*CoreV1Mock) Pods(namespace string) corev1.PodInterface {
	panic("unimplemented")
}

// PodTemplates implements v1.CoreV1Interface
func (*CoreV1Mock) PodTemplates(namespace string) corev1.PodTemplateInterface {
	panic("unimplemented")
}

// ReplicationControllers implements v1.CoreV1Interface
func (*CoreV1Mock) ReplicationControllers(namespace string) corev1.ReplicationControllerInterface {
	panic("unimplemented")
}

// ResourceQuotas implements v1.CoreV1Interface
func (*CoreV1Mock) ResourceQuotas(namespace string) corev1.ResourceQuotaInterface {
	panic("unimplemented")
}

// Secrets implements v1.CoreV1Interface
func (*CoreV1Mock) Secrets(namespace string) corev1.SecretInterface {
	return NewSecretMock()
}

// Services implements v1.CoreV1Interface
func (*CoreV1Mock) Services(namespace string) corev1.ServiceInterface {
	return NewServiceMock()
}

// ServiceAccounts implements v1.CoreV1Interface
func (*CoreV1Mock) ServiceAccounts(namespace string) corev1.ServiceAccountInterface {
	return NewServiceAccountMock()
}

// RESTClient implements v1.CoreV1Interface
func (*CoreV1Mock) RESTClient() rest.Interface {
	panic("unimplemented")
}
