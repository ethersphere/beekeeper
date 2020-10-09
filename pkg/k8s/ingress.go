package k8s

import (
	"context"
	"fmt"

	ev1b1 "k8s.io/api/extensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	ingressClassName string = "nginx-internal"
	ingressSpec             = ev1b1.IngressSpec{
		IngressClassName: &ingressClassName,
		Backend: &ev1b1.IngressBackend{
			ServiceName: name,
			ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
		},
		TLS: []ev1b1.IngressTLS{},
		Rules: []ev1b1.IngressRule{
			{
				Host:             "bee.beekeeper.staging.internal",
				IngressRuleValue: ev1b1.IngressRuleValue{HTTP: &ev1b1.HTTPIngressRuleValue{Paths: []ev1b1.HTTPIngressPath{{Backend: ev1b1.IngressBackend{ServiceName: name, ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: 80}}, Path: "/"}}}},
			},
		},
	}
	ingressAnnotations = map[string]string{
		"createdBy":                                          "beekeeper",
		"kubernetes.io/ingress.class":                        "nginx-internal",
		"nginx.ingress.kubernetes.io/affinity":               "cookie",
		"nginx.ingress.kubernetes.io/affinity-mode":          "persistent",
		"nginx.ingress.kubernetes.io/proxy-body-size":        "0",
		"nginx.ingress.kubernetes.io/proxy-read-timeout":     "7200",
		"nginx.ingress.kubernetes.io/proxy-send-timeout":     "7200",
		"nginx.ingress.kubernetes.io/session-cookie-max-age": "86400",
		"nginx.ingress.kubernetes.io/session-cookie-name":    "SWARMGATEWAY",
		"nginx.ingress.kubernetes.io/session-cookie-path":    "default",
		"nginx.ingress.kubernetes.io/ssl-redirect":           "true",
	}
)

// setIngress creates Ingress, if Ingress already exists does nothing
func setIngress(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, inSpec ev1b1.IngressSpec) (err error) {
	spec := &ev1b1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: ingressAnnotations,
			Labels:      labels,
		},
		Spec: inSpec,
	}

	_, err = clientset.ExtensionsV1beta1().Ingresses(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("ingress %s already exists in the namespace %s, updating the ingress\n", name, namespace)
			_, err = clientset.ExtensionsV1beta1().Ingresses(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}
