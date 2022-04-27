package mocks

import (
	v1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	rest "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.NetworkingV1Interface = (*NetworkingV1Mock)(nil)

type NetworkingV1Mock struct{}

func NewNetworkingV1MockMock() *NetworkingV1Mock {
	return &NetworkingV1Mock{}
}

// Ingresses implements v1.NetworkingV1Interface
func (*NetworkingV1Mock) Ingresses(namespace string) v1.IngressInterface {
	return NewIngressMock()
}

// IngressClasses implements v1.NetworkingV1Interface
func (*NetworkingV1Mock) IngressClasses() v1.IngressClassInterface {
	panic("unimplemented")
}

// NetworkPolicies implements v1.NetworkingV1Interface
func (*NetworkingV1Mock) NetworkPolicies(namespace string) v1.NetworkPolicyInterface {
	panic("unimplemented")
}

// RESTClient implements v1.NetworkingV1Interface
func (*NetworkingV1Mock) RESTClient() rest.Interface {
	panic("unimplemented")
}
