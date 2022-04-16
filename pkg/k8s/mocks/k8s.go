package mocks

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewForConfig(c *rest.Config) (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil
}
