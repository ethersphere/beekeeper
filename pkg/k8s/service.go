package k8s

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	svcSelector = map[string]string{
		"app.kubernetes.io/instance":   "bee",
		"app.kubernetes.io/name":       "bee",
		"app.kubernetes.io/managed-by": "beekeeper",
	}
	svcPorts = []v1.ServicePort{
		{
			Name:       "api",
			Protocol:   "TCP",
			Port:       80,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
		},
	}
	svcHeadlessPorts = []v1.ServicePort{
		{
			Name:       "api",
			Protocol:   "TCP",
			Port:       8080,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
		},
		{
			Name:       "p2p",
			Protocol:   "TCP",
			Port:       7070,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "p2p"},
		},
		{
			Name:       "debug",
			Protocol:   "TCP",
			Port:       6060,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "debug"},
		},
	}
)

// setService creates Service, if Service already exists does nothing
func setService(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, svcSpec v1.ServiceSpec) (err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: svcSpec,
	}
	_, err = clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("service %s already exists in the namespace %s\n", name, namespace)
			return nil
		}

		return err
	}

	return
}
