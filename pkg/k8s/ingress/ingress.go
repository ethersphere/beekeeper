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
	Backend     Backend
	TLS         []TLS
	Rules       []Rule
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
			Backend: func() *ev1b1.IngressBackend {
				b := o.Backend.toK8S()
				return &b
			}(),
			IngressClassName: func() *string {
				if len(o.Class) > 0 {
					return &o.Class
				}
				return nil
			}(),
			Rules: rulesToK8S(o.Rules),
			TLS:   tlsToK8S(o.TLS),
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

// Backend ...
type Backend struct {
	ServiceName string
	ServicePort string
}

func (b Backend) toK8S() (backend ev1b1.IngressBackend) {
	if len(b.ServiceName) > 0 {
		backend.ServiceName = b.ServiceName
	}
	if len(b.ServicePort) > 0 {
		backend.ServicePort = intstr.FromString(b.ServicePort)
	}
	return
}

// Rule ...
type Rule struct {
	Host  string
	Paths []Path
}

func (r Rule) toK8S() (rule ev1b1.IngressRule) {
	if len(r.Host) > 0 {
		rule.Host = r.Host
	}
	rule.IngressRuleValue = ev1b1.IngressRuleValue{
		HTTP: &ev1b1.HTTPIngressRuleValue{
			Paths: pathsToK8S(r.Paths),
		},
	}
	return
}

func rulesToK8S(list []Rule) (rules []ev1b1.IngressRule) {
	if len(list) > 0 {
		for _, r := range list {
			rules = append(rules, r.toK8S())
		}
	}
	return
}

// Path ...
type Path struct {
	Backend Backend
	Path    string
}

func (p Path) toK8S() (h ev1b1.HTTPIngressPath) {
	h.Backend = p.Backend.toK8S()
	if len(p.Path) > 0 {
		h.Path = p.Path
	}
	return
}

func pathsToK8S(list []Path) (paths []ev1b1.HTTPIngressPath) {
	if len(list) > 0 {
		for _, p := range list {
			paths = append(paths, p.toK8S())
		}
	}
	return
}

// TLS ...
type TLS struct {
	Hosts      []string
	SecretName string
}

func (t TLS) toK8S() (tls ev1b1.IngressTLS) {
	if len(t.Hosts) > 0 {
		for _, h := range t.Hosts {
			tls.Hosts = append(tls.Hosts, h)
		}
	}
	if len(t.SecretName) > 0 {
		tls.SecretName = t.SecretName
	}
	return
}

func tlsToK8S(list []TLS) (tls []ev1b1.IngressTLS) {
	if len(list) > 0 {
		for _, t := range list {
			tls = append(tls, t.toK8S())
		}
	}
	return
}
