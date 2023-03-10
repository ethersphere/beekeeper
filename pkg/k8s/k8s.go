package k8s

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

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

// ErrKubeconfigNotSet represents error when kubeconfig is empty string
var ErrKubeconfigNotSet = errors.New("kubeconfig is not set")

// Client manages communication with the Kubernetes
type Client struct {
	clientset kubernetes.Interface // Kubernetes client must handle authentication implicitly.
	logger    logging.Logger

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
	IngressRoute   *ingressroute.Client
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	InCluster      bool
	KubeconfigPath string
}

// ClientSetup holds functions for configuration of the Client.
// Functions are extracted for being able to mock them for unit tests.
type ClientSetup struct {
	NewForConfig         func(c *rest.Config) (*kubernetes.Clientset, error)
	InClusterConfig      func() (*rest.Config, error)
	BuildConfigFromFlags func(masterUrl string, kubeconfigPath string) (*rest.Config, error)
	FlagString           func(name string, value string, usage string) *string
	FlagParse            func()
	OsUserHomeDir        func() (string, error)
}

// NewClient returns Kubernetes clientset
func NewClient(s *ClientSetup, o *ClientOptions, logger logging.Logger) (c *Client, err error) {
	// set default options in case they are not provided
	if o == nil {
		o = &ClientOptions{
			InCluster:      false,
			KubeconfigPath: "~/.kube/config",
		}
	}

	// set in-cluster client
	if o.InCluster {
		config, err := s.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes in-cluster client config: %w", err)
		}

		clientset, err := s.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("creating Kubernetes in-cluster clientset: %w", err)
		}

		apiClientset, err := ingressroute.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("creating custom resource Kubernetes api in-cluster clientset: %w", err)
		}

		return newClient(clientset, apiClientset, logger), nil
	}

	// set client
	configPath := ""
	if len(o.KubeconfigPath) == 0 {
		return nil, ErrKubeconfigNotSet
	} else if o.KubeconfigPath == "~/.kube/config" {
		home, err := s.OsUserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("obtaining user's home dir: %w", err)
		}
		configPath = home + "/.kube/config"
	} else {
		configPath = o.KubeconfigPath
	}

	kubeconfig := s.FlagString("kubeconfig", configPath, "kubeconfig file")
	flag.Parse()

	config, err := s.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes client config: %w", err)
	}

	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(50, 200)

	// Wrap the default transport with our custom transport.
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return NewCustomTransport(rt, config)
	}

	clientset, err := s.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes clientset: %w", err)
	}

	apiClientset, err := ingressroute.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating custom resource Kubernetes api clientset: %w", err)
	}

	return newClient(clientset, apiClientset, logger), nil
}

// customTransport is an example custom transport that wraps the default transport
// and adds some custom behavior.
type customTransport struct {
	base         http.RoundTripper
	semaphore    chan struct{}
	qps          float64
	burst        int
	lastReqTime  time.Time
	requestCount int
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Acquire the semaphore to limit the number of concurrent requests.
	t.semaphore <- struct{}{}
	defer func() {
		<-t.semaphore
	}()

	// Limit the request rate using QPS and burst limits.
	now := time.Now()
	if t.requestCount >= t.burst || now.Sub(t.lastReqTime).Seconds() < 1/t.qps {
		time.Sleep(time.Duration(1/t.qps-now.Sub(t.lastReqTime).Seconds()) * time.Second)
		t.requestCount = 0
	}
	t.requestCount++
	t.lastReqTime = now

	// Forward the request to the base transport.
	resp, err := t.base.RoundTrip(req)
	// TODO retry?
	// if err != nil {
	// 	// retry
	// 	if strings.Contains(err.Error(), "context deadline exceeded (Client.Timeout exceeded while awaiting headers)") {
	// 		fmt.Printf("RETRY: %s", err.Error())
	// 		resp, err = t.base.RoundTrip(req)
	// 	} else {
	// 		fmt.Printf("ERROR: %s", err.Error())
	// 	}
	// }

	return resp, err
}

func NewCustomTransport(base http.RoundTripper, config *rest.Config) http.RoundTripper {
	qps := float64(config.QPS)
	burst := config.Burst
	return &customTransport{
		base:      base,
		semaphore: make(chan struct{}, 10),
		qps:       qps,
		burst:     burst,
	}
}

// newClient constructs a new *Client with the provided http Client, which
// should handle authentication implicitly, and sets all other services.
func newClient(clientset *kubernetes.Clientset, apiClientset *ingressroute.CustomResourceClient, logger logging.Logger) (c *Client) {
	c = &Client{
		clientset: clientset,
		logger:    logger,
	}

	c.ConfigMap = configmap.NewClient(clientset)
	c.Ingress = ingress.NewClient(clientset)
	c.Namespace = namespace.NewClient(clientset)
	c.Pods = pod.NewClient(clientset)
	c.PVC = persistentvolumeclaim.NewClient(clientset)
	c.Secret = secret.NewClient(clientset)
	c.ServiceAccount = serviceaccount.NewClient(clientset)
	c.Service = service.NewClient(clientset)
	c.StatefulSet = statefulset.NewClient(clientset)
	c.IngressRoute = ingressroute.NewClient(apiClientset)

	return c
}
