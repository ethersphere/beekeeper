package dynamick8s

import (
	"context"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	crdClient dynamic.ResourceInterface
}

// NewClient constructs a new dynamic Client.
func NewClient(kubeconfig string, namespace string, resource schema.GroupVersionResource) (c *Client, err error) {
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			kubeconfig = "kubeconfig"
		}
	}
	if namespace == "" {
		namespace = "default"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing kubeconfig file: %+v", err)
	}
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating k8s client: %+v", err)
	}
	crdClient := dynClient.Resource(resource)

	client := crdClient.Namespace(namespace)
	return newClient(client), nil
}

func newClient(crdClient dynamic.ResourceInterface) (c *Client) {
	c = &Client{crdClient: crdClient}
	return c
}

func (c *Client) Create(ctx context.Context, object *unstructured.Unstructured) (err error) {

	crdClient := c.crdClient
	_, err = crdClient.Create(ctx, object, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return nil
}

func (c *Client) Update(ctx context.Context, object *unstructured.Unstructured) (err error) {

	crdClient := c.crdClient
	_, err = crdClient.Update(ctx, object, metav1.UpdateOptions{})
	if err != nil {
		panic(err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, name string) (err error) {

	crdClient := c.crdClient
	err = crdClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		panic(err)
	}
	return nil
}
