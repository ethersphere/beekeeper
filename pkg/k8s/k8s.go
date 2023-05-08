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
	clientset            kubernetes.Interface    // Kubernetes client must handle authentication implicitly.
	logger               logging.Logger          // logger
	cs                   *ClientSetup            // ClientSetup holds functions for configuration of the Client.
	inCluster            bool                    // inCluster
	kubeconfigPath       string                  // kubeconfigPath
	rateLimiter          flowcontrol.RateLimiter // rateLimiter
	maxConcurentRequests int                     // maxConcurentRequests (semaphore)

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
func NewClient(options ...ClientOption) (c *Client, err error) {
	c = &Client{
		// set default values
		cs:                   NewClientSetup(),
		logger:               logging.New(io.Discard, 0, ""),
		inCluster:            false,
		kubeconfigPath:       "~/.kube/config",
		rateLimiter:          flowcontrol.NewTokenBucketRateLimiter(50, 100),
		maxConcurentRequests: 10,
	}

	// apply options
	for _, option := range options {
		option(c)
	}

	var config *rest.Config

	if c.inCluster {
		// set in-cluster client
		config, err = c.cs.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes in-cluster client config: %w", err)
		}
	} else {
		// set client from kubeconfig
		configPath := ""
		if len(c.kubeconfigPath) == 0 {
			return nil, ErrKubeconfigNotSet
		} else if c.kubeconfigPath == "~/.kube/config" {
			home, err := c.cs.OsUserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("obtaining user's home dir: %w", err)
			}
			configPath = home + "/.kube/config"
		} else {
			configPath = c.kubeconfigPath
		}

		kubeconfig := c.cs.FlagString("kubeconfig", configPath, "kubeconfig file")
		c.cs.FlagParse()

		config, err = c.cs.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes client config: %w", err)
		}
	}

	config.RateLimiter = c.rateLimiter

	// Wrap the default transport with our custom transport.
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return NewCustomTransport(rt, config, c.maxConcurentRequests)
	}

	clientset, err := c.cs.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes clientset: %w", err)
	}

	apiClientset, err := ingressroute.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating custom resource Kubernetes api clientset: %w", err)
	}

	c.setK8sClient(clientset, apiClientset)

	return c, nil
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all other services.
func (c *Client) setK8sClient(clientset kubernetes.Interface, apiClientset ingressroute.Interface) {
	c.clientset = clientset

	c.ConfigMap = configmap.NewClient(clientset)
	c.Ingress = ingress.NewClient(clientset)
	c.Namespace = namespace.NewClient(clientset)
	c.Pods = pod.NewClient(clientset)
	c.PVC = persistentvolumeclaim.NewClient(clientset)
	c.Secret = secret.NewClient(clientset)
	c.ServiceAccount = serviceaccount.NewClient(clientset)
	c.Service = service.NewClient(clientset)
	c.StatefulSet = statefulset.NewClient(clientset, c.logger)
	c.IngressRoute = ingressroute.NewClient(apiClientset)
}

// WithClientSetup sets the ClientSetup function, which is used for mocking.
func WithClientSetup(cs *ClientSetup) ClientOption {
	return func(c *Client) {
		c.cs = cs
	}
}

// WithLogger sets the logger
func WithLogger(logger logging.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithInCluster sets the inCluster
func WithInCluster(inCluster bool) ClientOption {
	return func(c *Client) {
		c.inCluster = inCluster
	}
}

// WithKubeconfigPath sets the kubeconfigPath
func WithKubeconfigPath(kubeconfigPath string) ClientOption {
	return func(c *Client) {
		c.kubeconfigPath = kubeconfigPath
	}
}

// WithRequestLimiter sets the rateLimiter
func WithRequestLimiter(rateLimiter flowcontrol.RateLimiter, maxConcurentRequests int) ClientOption {
	return func(c *Client) {
		c.rateLimiter = rateLimiter
		c.maxConcurentRequests = maxConcurentRequests
	}
}
