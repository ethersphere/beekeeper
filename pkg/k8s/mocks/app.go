package mocks

import (
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	rest "k8s.io/client-go/rest"
)

const (
	CreateBad string = "create_bad"
	UpdateBad string = "update_bad"
	DeleteBad string = "delete_bad"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.AppsV1Interface = (*AppV1)(nil)

type AppV1 struct{}

func NewAppV1() *AppV1 {
	return &AppV1{}
}

// ControllerRevisions implements v1.AppsV1Interface
func (*AppV1) ControllerRevisions(namespace string) v1.ControllerRevisionInterface {
	panic("unimplemented")
}

// DaemonSets implements v1.AppsV1Interface
func (*AppV1) DaemonSets(namespace string) v1.DaemonSetInterface {
	panic("unimplemented")
}

// Deployments implements v1.AppsV1Interface
func (*AppV1) Deployments(namespace string) v1.DeploymentInterface {
	panic("unimplemented")
}

// ReplicaSets implements v1.AppsV1Interface
func (*AppV1) ReplicaSets(namespace string) v1.ReplicaSetInterface {
	panic("unimplemented")
}

// StatefulSets implements v1.AppsV1Interface
func (*AppV1) StatefulSets(namespace string) v1.StatefulSetInterface {
	return NewStatefulSet(namespace)
}

// RESTClient implements v1.AppsV1Interface
func (*AppV1) RESTClient() rest.Interface {
	panic("unimplemented")
}
