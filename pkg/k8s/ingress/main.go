package ingress

import (
	"context"
	"fmt"

	ev1b1 "k8s.io/api/extensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// Options represents ingress' options
type Options struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Class       string
	Host        string
	ServiceName string
	ServicePort string
	Path        string
}

// Set creates Ingress, if Ingress already exists does nothing
func Set(ctx context.Context, clientset *kubernetes.Clientset, o Options) (err error) {
	spec := &ev1b1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.Name,
			Namespace:   o.Namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: ev1b1.IngressSpec{
			IngressClassName: &o.Class,
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

	_, err = clientset.ExtensionsV1beta1().Ingresses(o.Namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("ingress %s already exists in the namespace %s, updating the ingress\n", o.Name, o.Namespace)
			_, err = clientset.ExtensionsV1beta1().Ingresses(o.Namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}
