package mocks

import (
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	rest "k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.AppsV1Interface = (*AppV1Mock)(nil)

type AppV1Mock struct{}

func NewAppV1Mock() *AppV1Mock {
	return &AppV1Mock{}
}

// ControllerRevisions implements v1.AppsV1Interface
func (*AppV1Mock) ControllerRevisions(namespace string) v1.ControllerRevisionInterface {
	panic("unimplemented")
}

// DaemonSets implements v1.AppsV1Interface
func (*AppV1Mock) DaemonSets(namespace string) v1.DaemonSetInterface {
	panic("unimplemented")
}

// Deployments implements v1.AppsV1Interface
func (*AppV1Mock) Deployments(namespace string) v1.DeploymentInterface {
	panic("unimplemented")
}

// ReplicaSets implements v1.AppsV1Interface
func (*AppV1Mock) ReplicaSets(namespace string) v1.ReplicaSetInterface {
	panic("unimplemented")
}

// StatefulSets implements v1.AppsV1Interface
func (*AppV1Mock) StatefulSets(namespace string) v1.StatefulSetInterface {
	return NewStatefulSetMock(namespace)
}

// RESTClient implements v1.AppsV1Interface
func (*AppV1Mock) RESTClient() rest.Interface {
	panic("unimplemented")
}
