package k8s

import (
	"flag"
	"fmt"
	"os"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/namespace"
	"github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaim"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client manages communication with the Kubernetes
type Client struct {
	clientset *kubernetes.Clientset // Kubernetes client must handle authentication implicitly.

	// Services that K8S provides
	ConfigMap      *configmap.Client
	Ingress        *ingress.Client
	Namespace      *namespace.Client
	Pods           *pod.Client
	PVC            *persistentvolumeclaim.Client
	Secret         *secret.Client
	ServiceAccount *serviceaccount.Client
	Service        *service.Client
	StatefulSet    *statefulset.Client
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	InCluster      bool
	KubeconfigPath string
}

// NewClient returns Kubernetes clientset
func NewClient(o *ClientOptions) (c *Client, err error) {
	// set default options in case they are not provided
	if o == nil {
		o = &ClientOptions{
			InCluster:      false,
			KubeconfigPath: "~/.kube/config",
		}
	}

	// set in-cluster client
	if o.InCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes in-cluster client config: %w", err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes in-cluster clientset: %w", err)
		}

		return newClient(clientset), nil
	}

	// set client
	configPath := ""
	if len(o.KubeconfigPath) == 0 || o.KubeconfigPath == "~/.kube/config" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("obtaining user's home dir: %w", err)
		}
		configPath = home + "/.kube/config"
	} else {
		configPath = o.KubeconfigPath
	}

	kubeconfig := flag.String("kubeconfig", configPath, "kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes clientset: %w", err)
	}

	return newClient(clientset), nil
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all other services.
func newClient(clientset *kubernetes.Clientset) (c *Client) {
	c = &Client{clientset: clientset}

	c.ConfigMap = configmap.NewClient(clientset)
	c.Ingress = ingress.NewClient(clientset)
	c.Namespace = namespace.NewClient(clientset)
	c.Pods = pod.NewClient(clientset)
	c.PVC = persistentvolumeclaim.NewClient(clientset)
	c.Secret = secret.NewClient(clientset)
	c.ServiceAccount = serviceaccount.NewClient(clientset)
	c.Service = service.NewClient(clientset)
	c.StatefulSet = statefulset.NewClient(clientset)

	return c
}
