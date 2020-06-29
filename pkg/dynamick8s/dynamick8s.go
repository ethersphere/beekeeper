package dynamick8s

import (
	"context"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	crdClient dynamic.ResourceInterface
}

// NewClient constructs a new dynamic Client.
func NewClient(kubeconfig string, namespace string, resource schema.GroupVersionResource) (c *Client, err error) {
	var config *rest.Config

	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			kubeconfig = "kubeconfig"
		}
	}

	if kubeconfig == "incluster" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error parsing incluster kubeconfig: %+v", err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("error parsing kubeconfig file: %+v", err)
		}
	}
	if namespace == "" {
		namespace = "default"
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
		return fmt.Errorf("error creating k8s object: %+v", err)
	}
	return
}

func (c *Client) Update(ctx context.Context, object *unstructured.Unstructured) (err error) {

	crdClient := c.crdClient
	crd, err := crdClient.Get(ctx, object.GetName(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting resourceVersion k8s object: %+v", err)
	}
	object.SetResourceVersion(crd.GetResourceVersion())
	_, err = crdClient.Update(ctx, object, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating k8s object: %+v", err)
	}
	return
}

func (c *Client) Delete(ctx context.Context, name string) (err error) {

	crdClient := c.crdClient
	err = crdClient.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting k8s object: %+v", err)
	}
	return
}

func (c *Client) Get(ctx context.Context, name string) (resp *unstructured.Unstructured, err error) {

	crdClient := c.crdClient
	resp, err = crdClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating k8s object: %+v", err)
	}
	return resp, nil
}
