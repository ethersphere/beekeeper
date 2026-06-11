package mocks

import (
	v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.NetworkingV1Interface = (*NetworkingV1)(nil)

type NetworkingV1 struct {
	v1.NetworkingV1Interface
}

func NewNetworkingV1() *NetworkingV1 {
	return &NetworkingV1{}
}

// Ingresses implements v1.NetworkingV1Interface
func (*NetworkingV1) Ingresses(namespace string) v1.IngressInterface {
	return NewIngress()
}
