package k8s

import (
	"fmt"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
)

func TestNewClient(t *testing.T) {
	testTable := []struct {
		name     string
		options  *ClientOptions
		k8sFuncs ClientFunctions
		errorMsg error
	}{
		{
			name:     "in_cluster_config_error",
			options:  &ClientOptions{InCluster: true},
			errorMsg: fmt.Errorf("creating Kubernetes in-cluster client config: mock error"),
			k8sFuncs: ClientFunctions{
				NewForConfig:    mocks.NewClient(false).NewForConfig,
				InClusterConfig: mocks.NewClient(true).InClusterConfig,
			},
		},
		{
			name:     "in_cluster_clientset_error",
			options:  &ClientOptions{InCluster: true},
			errorMsg: fmt.Errorf("creating Kubernetes in-cluster clientset: mock error"),
			k8sFuncs: ClientFunctions{
				NewForConfig:    mocks.NewClient(true).NewForConfig,
				InClusterConfig: mocks.NewClient(false).InClusterConfig,
			},
		},
		{
			name:    "in_cluster",
			options: &ClientOptions{InCluster: true},
			k8sFuncs: ClientFunctions{
				NewForConfig:    mocks.NewClient(false).NewForConfig,
				InClusterConfig: mocks.NewClient(false).InClusterConfig,
			},
		},
		{
			name:    "not_in_cluster_default_path",
			options: nil,
			k8sFuncs: ClientFunctions{
				NewForConfig:         mocks.NewClient(false).NewForConfig,
				InClusterConfig:      mocks.NewClient(false).InClusterConfig,
				OsUserHomeDir:        mocks.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags: mocks.NewClient(false).BuildConfigFromFlags,
				FlagString:           mocks.FlagString,
				FlagParse:            mocks.FlagParse,
			},
		},
		{
			name:    "not_in_cluster_default_path_fail_clientset",
			options: nil,
			k8sFuncs: ClientFunctions{
				NewForConfig:         mocks.NewClient(true).NewForConfig,
				InClusterConfig:      mocks.NewClient(false).InClusterConfig,
				OsUserHomeDir:        mocks.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags: mocks.NewClient(false).BuildConfigFromFlags,
				FlagString:           mocks.FlagString,
				FlagParse:            mocks.FlagParse,
			},
			errorMsg: fmt.Errorf("creating Kubernetes clientset: mock error"),
		},
		{
			name:    "not_in_cluster_default_path_bad",
			options: nil,
			k8sFuncs: ClientFunctions{
				NewForConfig:         mocks.NewClient(false).NewForConfig,
				InClusterConfig:      mocks.NewClient(false).InClusterConfig,
				OsUserHomeDir:        mocks.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags: mocks.NewClient(true).BuildConfigFromFlags,
				FlagString:           mocks.FlagString,
				FlagParse:            mocks.FlagParse,
			},
			errorMsg: fmt.Errorf("creating Kubernetes client config: mock error"),
		},
		{
			name:    "not_in_cluster_other_path",
			options: &ClientOptions{InCluster: false, KubeconfigPath: "~/.kube/test_example"},
			k8sFuncs: ClientFunctions{
				NewForConfig:         mocks.NewClient(false).NewForConfig,
				InClusterConfig:      mocks.NewClient(false).InClusterConfig,
				BuildConfigFromFlags: mocks.NewClient(false).BuildConfigFromFlags,
				FlagString:           mocks.FlagString,
				FlagParse:            mocks.FlagParse,
			},
		},
		{
			name:    "not_in_cluster_fail_home_dir",
			options: &ClientOptions{InCluster: false, KubeconfigPath: "~/.kube/config"},
			k8sFuncs: ClientFunctions{
				NewForConfig:    mocks.NewClient(false).NewForConfig,
				InClusterConfig: mocks.NewClient(false).InClusterConfig,
				OsUserHomeDir:   mocks.NewClient(true).OsUserHomeDir,
			},
			errorMsg: fmt.Errorf("obtaining user's home dir: mock error"),
		},
		{
			name:     "not_in_cluster_empty_path",
			options:  &ClientOptions{},
			errorMsg: ErrKubeconfigNotSet,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			response, err := NewClient(test.k8sFuncs, test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Errorf("response expected, got nil")
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if response != nil {
					t.Errorf("response not expected, got: %v", response)
				}
			}
		})
	}
}
