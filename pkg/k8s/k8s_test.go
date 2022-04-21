package k8s

import (
	"fmt"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
)

func TestNewClient(t *testing.T) {
	testTable := []struct {
		name            string
		options         *ClientOptions
		newForConfig    NewForConfig
		inClusterConfig InClusterConfig
		errorMsg        error
	}{
		{
			name:            "in_cluster_config_error",
			options:         &ClientOptions{InCluster: true},
			errorMsg:        fmt.Errorf("creating Kubernetes in-cluster client config: mock error"),
			newForConfig:    mocks.NewClientMock(false).NewForConfig,
			inClusterConfig: mocks.NewClientMock(true).InClusterConfig,
		},
		{
			name:            "in_cluster_clientset_error",
			options:         &ClientOptions{InCluster: true},
			errorMsg:        fmt.Errorf("creating Kubernetes in-cluster clientset: mock error"),
			newForConfig:    mocks.NewClientMock(true).NewForConfig,
			inClusterConfig: mocks.NewClientMock(false).InClusterConfig,
		},
		{
			name:            "in_cluster",
			options:         &ClientOptions{InCluster: true},
			newForConfig:    mocks.NewClientMock(false).NewForConfig,
			inClusterConfig: mocks.NewClientMock(false).InClusterConfig,
		},
		{
			name:            "not_in_cluster_default_path",
			options:         nil,
			newForConfig:    mocks.NewClientMock(false).NewForConfig,
			inClusterConfig: mocks.NewClientMock(false).InClusterConfig,
		},
		// TODO panics when run in series, when run alone it works
		// {
		// 	name:            "not_in_cluster_default_path_bad",
		// 	options:         nil,
		// 	newForConfig:    mocks.NewClientMock(true).NewForConfig,
		// 	inClusterConfig: mocks.NewClientMock(false).InClusterConfig,
		// 	errorMsg:        fmt.Errorf("creating Kubernetes clientset: mock error"),
		// },
		// TODO panics when run in series, when run alone it works
		// {
		// 	name:            "not_in_cluster_other_path",
		// 	options:         &ClientOptions{InCluster: false, KubeconfigPath: "~/.kube/test_example"},
		// 	newForConfig:    mocks.NewClientMock(false).NewForConfig,
		// 	inClusterConfig: mocks.NewClientMock(false).InClusterConfig,
		// 	errorMsg:        fmt.Errorf("creating Kubernetes client config: CreateFile ~/.kube/test_example: The system cannot find the path specified."),
		// },
		{
			name:     "not_in_cluster_empty_path",
			options:  &ClientOptions{},
			errorMsg: ErrKubeconfigNotSet,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			response, err := NewClient(test.newForConfig, test.inClusterConfig, test.options)
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
