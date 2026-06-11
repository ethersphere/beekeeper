package mocks

import (
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// Clientset is a partial mock of kubernetes.Interface. Only the typed clients
// exercised by the tests are implemented; the embedded interface satisfies the
// rest and absorbs any methods added in future client-go versions. Calling an
// unimplemented client panics (nil-pointer dereference on the embedded value).
type Clientset struct {
	kubernetes.Interface
}

func NewClientset() *Clientset {
	return &Clientset{}
}

// AppsV1 implements kubernetes.Interface.
func (*Clientset) AppsV1() appsv1.AppsV1Interface {
	return NewAppV1()
}

// CoreV1 implements kubernetes.Interface.
func (*Clientset) CoreV1() corev1.CoreV1Interface {
	return NewCoreV1()
}

// NetworkingV1 implements kubernetes.Interface.
func (*Clientset) NetworkingV1() networkingv1.NetworkingV1Interface {
	return NewNetworkingV1()
}
