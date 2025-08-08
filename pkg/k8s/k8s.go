package k8s

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/namespace"
	"github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaim"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// ErrKubeconfigNotSet represents error when kubeconfig is empty string.
var ErrKubeconfigNotSet = errors.New("kubeconfig is not set")

// ClientOption holds optional parameters for the Client.
type ClientOption func(*Client)

// Client manages communication with the Kubernetes.
type Client struct {
	logger                logging.Logger
	clientConfig          *ClientConfig
	inCluster             bool
	kubeconfigPath        string
	rateLimiter           flowcontrol.RateLimiter
	maxConcurrentRequests int

	// exported services that K8S provides
	ConfigMap      *configmap.Client
	Ingress        *ingress.Client
	Namespace      *namespace.Client
	Pods           *pod.Client
	PVC            *persistentvolumeclaim.Client
	Secret         *secret.Client
	ServiceAccount *serviceaccount.Client
	Service        *service.Client
	StatefulSet    *statefulset.Client
	IngressRoute   *ingressroute.Client
}

// NewClient returns a new Kubernetes client.
func NewClient(opts ...ClientOption) (c *Client, err error) {
	c = &Client{
		// set default values
		clientConfig:          newClientConfig(),
		logger:                logging.New(io.Discard, 0),
		inCluster:             false,
		kubeconfigPath:        "~/.kube/config",
		rateLimiter:           flowcontrol.NewTokenBucketRateLimiter(50, 100),
		maxConcurrentRequests: 20,
	}

	// apply options
	for _, option := range opts {
		option(c)
	}

	var config *rest.Config

	if c.inCluster {
		// set in-cluster client
		config, err = c.clientConfig.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes in-cluster client config: %w", err)
		}
	} else {
		// set client from kubeconfig
		configPath := ""
		if len(c.kubeconfigPath) == 0 {
			return nil, ErrKubeconfigNotSet
		} else if c.kubeconfigPath == "~/.kube/config" {
			home, err := c.clientConfig.OsUserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("obtaining user's home dir: %w", err)
			}
			configPath = home + "/.kube/config"
		} else {
			configPath = c.kubeconfigPath
		}

		kubeconfig := c.clientConfig.FlagString("kubeconfig", configPath, "kubeconfig file")
		c.clientConfig.FlagParse()

		config, err = c.clientConfig.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes client config: %w", err)
		}
	}

	config.RateLimiter = c.rateLimiter
	semaphore := make(chan struct{}, c.maxConcurrentRequests)

	ct := NewCustomTransport(config, semaphore, c.logger)

	// Wrap the default transport with our custom transport.
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return ct.SetBaseTransport(rt)
	})

	clientset, err := c.clientConfig.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes clientset: %w", err)
	}

	apiClientset, err := c.clientConfig.NewIngressRouteClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating custom resource Kubernetes api clientset: %w", err)
	}

	c.setK8sClient(clientset, apiClientset)

	return c, nil
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all other services.
func (c *Client) setK8sClient(clientset kubernetes.Interface, apiClientset ingressroute.Interface) {
	c.ConfigMap = configmap.NewClient(clientset)
	c.Ingress = ingress.NewClient(clientset, c.logger)
	c.Namespace = namespace.NewClient(clientset)
	c.Pods = pod.NewClient(clientset, c.logger)
	c.PVC = persistentvolumeclaim.NewClient(clientset)
	c.Secret = secret.NewClient(clientset)
	c.ServiceAccount = serviceaccount.NewClient(clientset)
	c.Service = service.NewClient(clientset)
	c.StatefulSet = statefulset.NewClient(clientset, c.logger)
	c.IngressRoute = ingressroute.NewClient(apiClientset, c.logger)
}

// WithMockClientConfig sets the ClientConfig function, which is used for only when mocking.
func WithMockClientConfig(cs *ClientConfig) ClientOption {
	return func(c *Client) {
		if cs != nil {
			c.clientConfig = cs
		}
	}
}

// WithLogger sets the logger for the Client.
func WithLogger(logger logging.Logger) ClientOption {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithInCluster sets the inCluster flag for the Client.
func WithInCluster(inCluster bool) ClientOption {
	return func(c *Client) {
		c.inCluster = inCluster
	}
}

// WithKubeconfigPath sets the kubeconfigPath for the Client.
func WithKubeconfigPath(kubeconfigPath string) ClientOption {
	return func(c *Client) {
		c.kubeconfigPath = kubeconfigPath
	}
}

// WithRequestLimiter sets the rateLimiter and maxConcurentRequests for the Client.
func WithRequestLimiter(rateLimiter flowcontrol.RateLimiter, maxConcurentRequests int) ClientOption {
	return func(c *Client) {
		if rateLimiter != nil {
			c.rateLimiter = rateLimiter
		}
		if maxConcurentRequests >= 0 {
			c.maxConcurrentRequests = maxConcurentRequests
		}
	}
}
