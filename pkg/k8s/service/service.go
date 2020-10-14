package service

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Service.
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient constructs a new Client.
func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations map[string]string
	Labels      map[string]string
	Ports       []Port
	Selector    map[string]string
	Type        string
}

// Port represents service's port
type Port struct {
	Name       string
	Protocol   string
	Port       int32
	TargetPort string
	Nodeport   int32
}

// Set creates Service, if Service already exists does nothing
func (c Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports:    k8sPorts(o.Ports),
			Selector: o.Selector,
			Type:     v1.ServiceType(o.Type),
		},
	}
	_, err = c.clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			fmt.Printf("service %s already exists in the namespace %s\n", name, namespace)
			return nil
		}

		return err
	}

	return
}

func (p Port) toK8S() v1.ServicePort {
	return v1.ServicePort{
		Name:       p.Name,
		Protocol:   v1.Protocol(p.Protocol),
		Port:       p.Port,
		TargetPort: intstr.FromString(p.TargetPort),
		NodePort:   p.Nodeport,
	}

}
func k8sPorts(ports []Port) (ks []v1.ServicePort) {
	for _, port := range ports {
		ks = append(ks, port.toK8S())
	}
	return
}
