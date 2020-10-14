package k8s

import (
	"flag"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/namespace"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/services"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client manages communication with the Kubernetes
type Client struct {
	clientset *kubernetes.Clientset

	// Services that K8S provides
	ConfigMap      *configmap.Service
	Ingress        *ingress.Service
	Namespace      *namespace.Service
	Secret         *secret.Service
	ServiceAccount *serviceaccount.Service
	Services       *services.Service
	Stateful       *statefulset.Service
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	KubeconfigPath string
}

// NewClient returns Kubernetes clientset
func NewClient(o *ClientOptions) (c *Client) {
	if o == nil {
		o = &ClientOptions{
			KubeconfigPath: "~/.kube/config",
		}
	}

	kubeconfig := flag.String("kubeconfig", o.KubeconfigPath, "kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	return newClient(kubernetes.NewForConfigOrDie(config))
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all other services.
func newClient(clientset *kubernetes.Clientset) (c *Client) {
	c = &Client{clientset: clientset}

	c.ConfigMap = configmap.NewService(clientset)
	c.Ingress = ingress.NewService(clientset)
	c.Namespace = namespace.NewService(clientset)
	c.Secret = secret.NewService(clientset)
	c.ServiceAccount = serviceaccount.NewService(clientset)
	c.Services = services.NewService(clientset)
	c.Stateful = statefulset.NewService(clientset)

	return c
}
