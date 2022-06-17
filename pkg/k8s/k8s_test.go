package k8s

import (
	"fmt"
	"io"
	"testing"

	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func TestNewClient(t *testing.T) {
	testTable := []struct {
		name     string
		options  *ClientOptions
		k8sFuncs *ClientSetup
		errorMsg error
	}{
		{
			name:     "in_cluster_config_error",
			options:  &ClientOptions{InCluster: true},
			errorMsg: fmt.Errorf("creating Kubernetes in-cluster client config: mock error"),
			k8sFuncs: &ClientSetup{
				NewForConfig:    mock.NewClient(false).NewForConfig,
				InClusterConfig: mock.NewClient(true).InClusterConfig,
			},
		},
		{
			name:     "in_cluster_clientset_error",
			options:  &ClientOptions{InCluster: true},
			errorMsg: fmt.Errorf("creating Kubernetes in-cluster clientset: mock error"),
			k8sFuncs: &ClientSetup{
				NewForConfig:    mock.NewClient(true).NewForConfig,
				InClusterConfig: mock.NewClient(false).InClusterConfig,
			},
		},
		{
			name:    "in_cluster",
			options: &ClientOptions{InCluster: true},
			k8sFuncs: &ClientSetup{
				NewForConfig:    mock.NewClient(false).NewForConfig,
				InClusterConfig: mock.NewClient(false).InClusterConfig,
			},
		},
		{
			name:    "not_in_cluster_default_path",
			options: nil,
			k8sFuncs: &ClientSetup{
				NewForConfig:         mock.NewClient(false).NewForConfig,
				InClusterConfig:      mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:        mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags: mock.NewClient(false).BuildConfigFromFlags,
				FlagString:           mock.FlagString,
				FlagParse:            mock.FlagParse,
			},
		},
		{
			name:    "not_in_cluster_default_path_fail_clientset",
			options: nil,
			k8sFuncs: &ClientSetup{
				NewForConfig:         mock.NewClient(true).NewForConfig,
				InClusterConfig:      mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:        mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags: mock.NewClient(false).BuildConfigFromFlags,
				FlagString:           mock.FlagString,
				FlagParse:            mock.FlagParse,
			},
			errorMsg: fmt.Errorf("creating Kubernetes clientset: mock error"),
		},
		{
			name:    "not_in_cluster_default_path_bad",
			options: nil,
			k8sFuncs: &ClientSetup{
				NewForConfig:         mock.NewClient(false).NewForConfig,
				InClusterConfig:      mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:        mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags: mock.NewClient(true).BuildConfigFromFlags,
				FlagString:           mock.FlagString,
				FlagParse:            mock.FlagParse,
			},
			errorMsg: fmt.Errorf("creating Kubernetes client config: mock error"),
		},
		{
			name:    "not_in_cluster_other_path",
			options: &ClientOptions{InCluster: false, KubeconfigPath: "~/.kube/test_example"},
			k8sFuncs: &ClientSetup{
				NewForConfig:         mock.NewClient(false).NewForConfig,
				InClusterConfig:      mock.NewClient(false).InClusterConfig,
				BuildConfigFromFlags: mock.NewClient(false).BuildConfigFromFlags,
				FlagString:           mock.FlagString,
				FlagParse:            mock.FlagParse,
			},
		},
		{
			name:    "not_in_cluster_fail_home_dir",
			options: &ClientOptions{InCluster: false, KubeconfigPath: "~/.kube/config"},
			k8sFuncs: &ClientSetup{
				NewForConfig:    mock.NewClient(false).NewForConfig,
				InClusterConfig: mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:   mock.NewClient(true).OsUserHomeDir,
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
			response, err := NewClient(test.k8sFuncs, test.options, logging.New(io.Discard, 0))
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
