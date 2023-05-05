package k8s

import (
	"flag"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientSetupOption func(*ClientSetup)

// NewClientSetup returns a new ClientSetup.
func NewClientSetup(options ...ClientSetupOption) *ClientSetup {
	cs := &ClientSetup{
		NewForConfig:         kubernetes.NewForConfig,
		InClusterConfig:      rest.InClusterConfig,
		BuildConfigFromFlags: clientcmd.BuildConfigFromFlags,
		FlagString:           flag.String,
		FlagParse:            flag.Parse,
		OsUserHomeDir:        os.UserHomeDir,
	}

	for _, option := range options {
		option(cs)
	}

	return cs
}

// ClientSetup holds functions for configuration of the Client.
// Functions are extracted for being able to mock them for unit tests.
type ClientSetup struct {
	NewForConfig         func(c *rest.Config) (*kubernetes.Clientset, error)                 // kubernetes.NewForConfig
	InClusterConfig      func() (*rest.Config, error)                                        // rest.InClusterConfig
	BuildConfigFromFlags func(masterUrl string, kubeconfigPath string) (*rest.Config, error) // clientcmd.BuildConfigFromFlags
	FlagString           func(name string, value string, usage string) *string               // flag.String
	FlagParse            func()                                                              // flag.Parse
	OsUserHomeDir        func() (string, error)                                              // os.UserHomeDir
}

// WithNewForConfig sets the NewForConfig function.
func WithNewForConfig(f func(c *rest.Config) (*kubernetes.Clientset, error)) ClientSetupOption {
	return func(cs *ClientSetup) {
		cs.NewForConfig = f
	}
}

// WithInClusterConfig sets the InClusterConfig function.
func WithInClusterConfig(f func() (*rest.Config, error)) ClientSetupOption {
	return func(cs *ClientSetup) {
		cs.InClusterConfig = f
	}
}

// WithBuildConfigFromFlags sets the BuildConfigFromFlags function.
func WithBuildConfigFromFlags(f func(masterUrl string, kubeconfigPath string) (*rest.Config, error)) ClientSetupOption {
	return func(cs *ClientSetup) {
		cs.BuildConfigFromFlags = f
	}
}

// WithFlagString sets the FlagString function.
func WithFlagString(f func(name string, value string, usage string) *string) ClientSetupOption {
	return func(cs *ClientSetup) {
		cs.FlagString = f
	}
}

// WithFlagParse sets the FlagParse function.
func WithFlagParse(f func()) ClientSetupOption {
	return func(cs *ClientSetup) {
		cs.FlagParse = f
	}
}

// WithOsUserHomeDir sets the OsUserHomeDir function.
func WithOsUserHomeDir(f func() (string, error)) ClientSetupOption {
	return func(cs *ClientSetup) {
		cs.OsUserHomeDir = f
	}
}
