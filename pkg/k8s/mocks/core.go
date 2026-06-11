package mocks

import (
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ corev1.CoreV1Interface = (*CoreV1)(nil)

type CoreV1 struct {
	corev1.CoreV1Interface
}

func NewCoreV1() *CoreV1 {
	return &CoreV1{}
}

// ConfigMaps implements v1.CoreV1Interface
func (*CoreV1) ConfigMaps(namespace string) corev1.ConfigMapInterface {
	return NewConfigMap()
}

// Namespaces implements v1.CoreV1Interface
func (*CoreV1) Namespaces() corev1.NamespaceInterface {
	return NewNamespace()
}

// PersistentVolumeClaims implements v1.CoreV1Interface
func (*CoreV1) PersistentVolumeClaims(namespace string) corev1.PersistentVolumeClaimInterface {
	return NewPvc()
}

// Pods implements v1.CoreV1Interface
func (*CoreV1) Pods(namespace string) corev1.PodInterface {
	return NewPod()
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
