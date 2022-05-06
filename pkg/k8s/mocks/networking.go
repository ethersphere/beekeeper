package mocks

import (
	v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	rest "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.NetworkingV1Interface = (*NetworkingV1)(nil)

type NetworkingV1 struct{}

func NewNetworkingV1() *NetworkingV1 {
	return &NetworkingV1{}
}

// Ingresses implements v1.NetworkingV1Interface
func (*NetworkingV1) Ingresses(namespace string) v1.IngressInterface {
	return NewIngress()
}

// IngressClasses implements v1.NetworkingV1Interface
func (*NetworkingV1) IngressClasses() v1.IngressClassInterface {
	panic("unimplemented")
}

// NetworkPolicies implements v1.NetworkingV1Interface
func (*NetworkingV1) NetworkPolicies(namespace string) v1.NetworkPolicyInterface {
	panic("unimplemented")
}

// RESTClient implements v1.NetworkingV1Interface
func (*NetworkingV1) RESTClient() rest.Interface {
	panic("unimplemented")
}
