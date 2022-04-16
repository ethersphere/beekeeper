package k8s

import (
	"fmt"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
	"k8s.io/client-go/kubernetes"
)

func TestNewClient(t *testing.T) {
	testTable := []struct {
		name         string
		options      *ClientOptions
		newForConfig NewForConfig
		errorMsg     error
	}{
		{
			name:         "default",
			options:      nil,
			errorMsg:     fmt.Errorf("creating Kubernetes clientset: no Auth Provider found for name \"oidc\""),
			newForConfig: kubernetes.NewForConfig,
		},
		{
			name:         "options_in_cluster_default_path",
			options:      &ClientOptions{InCluster: true, KubeconfigPath: "~/.kube/config"},
			errorMsg:     fmt.Errorf("creating Kubernetes in-cluster client config: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined"),
			newForConfig: kubernetes.NewForConfig,
		},
		{
			name:         "options_in_cluster_empty_path",
			options:      &ClientOptions{InCluster: true, KubeconfigPath: ""},
			errorMsg:     fmt.Errorf("creating Kubernetes in-cluster client config: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined"),
			newForConfig: kubernetes.NewForConfig,
		},
		// TODO fails to execute
		// {
		// 	name:         "options_not_in_cluster_different_path",
		// 	options:      &ClientOptions{InCluster: false, KubeconfigPath: "~/.kube/unit_test"},
		// 	errorMsg:     fmt.Errorf("creating Kubernetes in-cluster client config: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined"),
		// 	newForConfig: kubernetes.NewForConfig,
		// },
		{
			name:         "options_not_in_cluster_empty_path",
			options:      &ClientOptions{},
			errorMsg:     ErrKubeconfigNotSet,
			newForConfig: kubernetes.NewForConfig,
		},
		// TODO chek why test fails when series are run; if it is run alone, it works
		{
			name:         "default_mock_new_for_config",
			options:      nil,
			newForConfig: mocks.NewForConfig,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			response, err := NewClient(test.newForConfig, test.options)
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
