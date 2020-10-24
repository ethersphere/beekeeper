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
	Annotations                   map[string]string
	Labels                        map[string]string
	ClusterIP                     string
	ExternalIPs                   []string
	ExternalName                  string
	ExternalTrafficPolicy         string
	LoadBalancerIP                string
	LoadBalancerSourceRanges      []string
	Ports                         []Port
	PublishNotReadyAddresses      bool
	Selector                      map[string]string
	SessionAffinity               string
	SessionAffinityTimeoutSeconds int32
	TopologyKeys                  []string
	Type                          string
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
			ClusterIP:                o.ClusterIP,
			ExternalIPs:              o.ExternalIPs,
			ExternalName:             o.ExternalName,
			ExternalTrafficPolicy:    v1.ServiceExternalTrafficPolicyType(o.ExternalTrafficPolicy),
			LoadBalancerIP:           o.LoadBalancerIP,
			LoadBalancerSourceRanges: o.LoadBalancerSourceRanges,
			Ports:                    k8sPorts(o.Ports),
			PublishNotReadyAddresses: o.PublishNotReadyAddresses,
			Selector:                 o.Selector,
			SessionAffinity:          v1.ServiceAffinity(o.SessionAffinity),
			SessionAffinityConfig: func() *v1.SessionAffinityConfig {
				if o.SessionAffinityTimeoutSeconds > 0 {
					return &v1.SessionAffinityConfig{
						ClientIP: &v1.ClientIPConfig{
							TimeoutSeconds: &o.SessionAffinityTimeoutSeconds,
						},
					}
				}
				return nil
			}(),
			TopologyKeys: o.TopologyKeys,
			Type:         v1.ServiceType(o.Type),
		},
	}
	_, err = c.clientset.CoreV1().Services(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		// TODO: fix condition, always true
		if !errors.IsNotFound(err) {
			fmt.Printf("service %s already exists in the namespace %s\n", name, namespace)
			return nil
		}
		return
	}

	return
}

// Port represents service's port
type Port struct {
	Name        string
	AppProtocol string
	Nodeport    int32
	Protocol    string
	Port        int32
	TargetPort  string
}

func (p Port) toK8S() v1.ServicePort {
	return v1.ServicePort{
		Name:     p.Name,
		Protocol: v1.Protocol(p.Protocol),
		AppProtocol: func() *string {
			if len(p.AppProtocol) > 0 {
				return &p.AppProtocol
			}
			return nil
		}(),
		Port: p.Port,
		TargetPort: func() intstr.IntOrString {
			if len(p.TargetPort) > 0 {
				return intstr.FromString(p.TargetPort)
			}
			return intstr.IntOrString{}
		}(),
		NodePort: p.Nodeport,
	}

}

func k8sPorts(ports []Port) (ks []v1.ServicePort) {
	for _, port := range ports {
		ks = append(ks, port.toK8S())
	}
	return
}
