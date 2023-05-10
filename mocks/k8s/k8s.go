package mocks

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// compile simulation whether ClientsetMock implements interface
var _ kubernetes.Interface = (*Clientset)(nil)

type Client struct {
	expectError bool
}

func NewClient(expectError bool) *Client {
	return &Client{expectError: expectError}
}

// NewForConfig returns a new Kubernetes clientset
func (c *Client) NewForConfig(*rest.Config) (*kubernetes.Clientset, error) {
	if c.expectError {
		return nil, fmt.Errorf("mock error")
	}
	return &kubernetes.Clientset{}, nil
}

// NewIngressRouteClientForConfig returns a new ingressroute client
func (c *Client) NewIngressRouteClientForConfig(*rest.Config) (*ingressroute.CustomResourceClient, error) {
	if c.expectError {
		return nil, fmt.Errorf("mock error")
	}
	return &ingressroute.CustomResourceClient{}, nil
}

func (c *Client) InClusterConfig() (*rest.Config, error) {
	if c.expectError {
		return nil, fmt.Errorf("mock error")
	}
	return &rest.Config{}, nil
}

func (c *Client) BuildConfigFromFlags(masterUrl string, kubeconfigPath string) (*rest.Config, error) {
	if c.expectError {
		return nil, fmt.Errorf("mock error")
	}
	return &rest.Config{}, nil
}

func (c *Client) OsUserHomeDir() (string, error) {
	if c.expectError {
		return "", fmt.Errorf("mock error")
	}
	return "home", nil
}

func FlagString(name string, value string, usage string) *string {
	return new(string)
}

func FlagParse() {}
