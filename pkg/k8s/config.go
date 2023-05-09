package k8s

import (
	"flag"
	"os"

	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// newClientConfig returns a new default ClienConfig.
func newClientConfig() *ClientConfig {
	return &ClientConfig{
		NewForConfig:                   kubernetes.NewForConfig,
		NewIngressRouteClientForConfig: ingressroute.NewForConfig,
		InClusterConfig:                rest.InClusterConfig,
		BuildConfigFromFlags:           clientcmd.BuildConfigFromFlags,
		FlagString:                     flag.String,
		FlagParse:                      flag.Parse,
		OsUserHomeDir:                  os.UserHomeDir,
	}
}

// ClientConfig holds functions for configration of the Kubernetes client.
// Functions are extracted to be able to mock them in tests.
type ClientConfig struct {
	NewForConfig                   func(c *rest.Config) (*kubernetes.Clientset, error)                 // kubernetes.NewForConfig
	NewIngressRouteClientForConfig func(c *rest.Config) (*ingressroute.CustomResourceClient, error)    // ingressroute.NewForConfig
	InClusterConfig                func() (*rest.Config, error)                                        // rest.InClusterConfig
	BuildConfigFromFlags           func(masterUrl string, kubeconfigPath string) (*rest.Config, error) // clientcmd.BuildConfigFromFlags
	FlagString                     func(name string, value string, usage string) *string               // flag.String
	FlagParse                      func()                                                              // flag.Parse
	OsUserHomeDir                  func() (string, error)                                              // os.UserHomeDir
}
