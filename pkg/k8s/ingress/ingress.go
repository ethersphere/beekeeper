package ingress

import (
	"context"
	"fmt"

	ev1b1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes Ingress.
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
	Class       string
	Host        string
	ServiceName string
	ServicePort string
	Path        string
}

// Set creates Ingress, if Ingress already exists does nothing
func (c Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &ev1b1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: ev1b1.IngressSpec{
			Backend: &ev1b1.IngressBackend{
				ServiceName: o.ServiceName,
				ServicePort: intstr.FromString(o.ServicePort),
			},
			TLS: []ev1b1.IngressTLS{},
			Rules: []ev1b1.IngressRule{
				{
					Host: o.Host,
					IngressRuleValue: ev1b1.IngressRuleValue{
						HTTP: &ev1b1.HTTPIngressRuleValue{
							Paths: []ev1b1.HTTPIngressPath{{
								Backend: ev1b1.IngressBackend{
									ServiceName: o.ServiceName,
									ServicePort: intstr.FromString(o.ServicePort),
								},
								Path: o.Path,
							}},
						},
					},
				},
			},
		},
	}

	_, err = c.clientset.ExtensionsV1beta1().Ingresses(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			fmt.Printf("ingress %s already exists in the namespace %s, updating the ingress\n", name, namespace)
			_, err = c.clientset.ExtensionsV1beta1().Ingresses(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}
