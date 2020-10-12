package service

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// Options represents service's options
type Options struct {
	Name        string
	Namespace   string
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

func (p Port) toK8S() v1.ServicePort {
	return v1.ServicePort{
		Name:       p.Name,
		Protocol:   v1.Protocol(p.Protocol),
		Port:       p.Port,
		TargetPort: intstr.FromString(p.TargetPort),
		NodePort:   p.Nodeport,
	}

}

// Set creates Service, if Service already exists does nothing
func Set(ctx context.Context, clientset *kubernetes.Clientset, o Options) (err error) {
	spec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.Name,
			Namespace:   o.Namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports:    k8sPorts(o.Ports),
			Selector: o.Selector,
			Type:     v1.ServiceType(o.Type),
		},
	}
	_, err = clientset.CoreV1().Services(o.Namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("service %s already exists in the namespace %s\n", o.Name, o.Namespace)
			return nil
		}

		return err
	}

	return
}

func k8sPorts(ports []Port) (ks []v1.ServicePort) {
	for _, port := range ports {
		ks = append(ks, port.toK8S())
	}
	return
}
