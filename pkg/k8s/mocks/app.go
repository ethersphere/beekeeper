package mocks

import (
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

const (
	CreateBad string = "create_bad"
	UpdateBad string = "update_bad"
	DeleteBad string = "delete_bad"
)

// compile simulation whether ClientsetMock implements interface
var _ v1.AppsV1Interface = (*AppV1)(nil)

type AppV1 struct {
	v1.AppsV1Interface
}

func NewAppV1() *AppV1 {
	return &AppV1{}
}

// StatefulSets implements v1.AppsV1Interface
func (*AppV1) StatefulSets(namespace string) v1.StatefulSetInterface {
	return NewStatefulSet(namespace)
}
