package k8s_test

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func TestNewClient(t *testing.T) {
	testTable := []struct {
		name      string
		options   []k8s.ClientOption
		k8sConfig *k8s.ClientConfig
		errorMsg  error
	}{
		{
			name:     "in_cluster_config_error",
			options:  []k8s.ClientOption{k8s.WithInCluster(true)},
			errorMsg: fmt.Errorf("creating Kubernetes in-cluster client config: mock error"),
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:    mock.NewClient(false).NewForConfig,
				InClusterConfig: mock.NewClient(true).InClusterConfig,
			},
		},
		{
			name:     "in_cluster_clientset_error",
			options:  []k8s.ClientOption{k8s.WithInCluster(true)},
			errorMsg: fmt.Errorf("creating Kubernetes clientset: mock error"),
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:    mock.NewClient(true).NewForConfig,
				InClusterConfig: mock.NewClient(false).InClusterConfig,
			},
		},
		{
			name:    "in_cluster",
			options: []k8s.ClientOption{k8s.WithInCluster(true)},
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:                   mock.NewClient(false).NewForConfig,
				InClusterConfig:                mock.NewClient(false).InClusterConfig,
				NewIngressRouteClientForConfig: mock.NewClient(false).NewIngressRouteClientForConfig,
			},
		},
		{
			name:    "not_in_cluster_default_path",
			options: nil,
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:                   mock.NewClient(false).NewForConfig,
				NewIngressRouteClientForConfig: mock.NewClient(false).NewIngressRouteClientForConfig,
				InClusterConfig:                mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:                  mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags:           mock.NewClient(false).BuildConfigFromFlags,
				FlagString:                     mock.FlagString,
				FlagParse:                      mock.FlagParse,
			},
		},
		{
			name:    "not_in_cluster_default_path_fail_clientset",
			options: nil,
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:                   mock.NewClient(true).NewForConfig,
				NewIngressRouteClientForConfig: mock.NewClient(false).NewIngressRouteClientForConfig,
				InClusterConfig:                mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:                  mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags:           mock.NewClient(false).BuildConfigFromFlags,
				FlagString:                     mock.FlagString,
				FlagParse:                      mock.FlagParse,
			},
			errorMsg: fmt.Errorf("creating Kubernetes clientset: mock error"),
		},
		{
			name:    "not_in_cluster_default_path_fail_ingressroute_client",
			options: nil,
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:                   mock.NewClient(false).NewForConfig,
				NewIngressRouteClientForConfig: mock.NewClient(true).NewIngressRouteClientForConfig,
				InClusterConfig:                mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:                  mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags:           mock.NewClient(false).BuildConfigFromFlags,
				FlagString:                     mock.FlagString,
				FlagParse:                      mock.FlagParse,
			},
			errorMsg: fmt.Errorf("creating custom resource Kubernetes api clientset: mock error"),
		},
		{
			name:    "not_in_cluster_default_path_bad",
			options: nil,
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:                   mock.NewClient(false).NewForConfig,
				NewIngressRouteClientForConfig: mock.NewClient(false).NewIngressRouteClientForConfig,
				InClusterConfig:                mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:                  mock.NewClient(false).OsUserHomeDir,
				BuildConfigFromFlags:           mock.NewClient(true).BuildConfigFromFlags,
				FlagString:                     mock.FlagString,
				FlagParse:                      mock.FlagParse,
			},
			errorMsg: fmt.Errorf("creating Kubernetes client config: mock error"),
		},
		{
			name:    "not_in_cluster_other_path",
			options: []k8s.ClientOption{k8s.WithInCluster(false), k8s.WithKubeconfigPath("~/.kube/test_example")},
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:                   mock.NewClient(false).NewForConfig,
				NewIngressRouteClientForConfig: mock.NewClient(false).NewIngressRouteClientForConfig,
				InClusterConfig:                mock.NewClient(false).InClusterConfig,
				BuildConfigFromFlags:           mock.NewClient(false).BuildConfigFromFlags,
				FlagString:                     mock.FlagString,
				FlagParse:                      mock.FlagParse,
			},
		},
		{
			name:    "not_in_cluster_fail_home_dir",
			options: []k8s.ClientOption{k8s.WithInCluster(false), k8s.WithKubeconfigPath("~/.kube/config")},
			k8sConfig: &k8s.ClientConfig{
				NewForConfig:    mock.NewClient(false).NewForConfig,
				InClusterConfig: mock.NewClient(false).InClusterConfig,
				OsUserHomeDir:   mock.NewClient(true).OsUserHomeDir,
			},
			errorMsg: fmt.Errorf("obtaining user's home dir: mock error"),
		},
		{
			name:     "not_in_cluster_empty_path",
			options:  []k8s.ClientOption{k8s.WithInCluster(false), k8s.WithKubeconfigPath("")},
			errorMsg: k8s.ErrKubeconfigNotSet,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			test.options = append(test.options, k8s.WithMockClientConfig(test.k8sConfig))
			response, err := k8s.NewClient(test.options...)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Errorf("response expected, got nil")
				}

				// Get the value of the struct using reflection.
				val := reflect.ValueOf(response)
				val = val.Elem()

				// Iterate over the fields of the struct.
				for i := 0; i < val.NumField(); i++ {
					// Get the value of the field.
					fieldVal := val.Field(i)

					// Check if the field is nil.
					if (fieldVal.Kind() == reflect.Interface || fieldVal.Kind() == reflect.Ptr) && fieldVal.IsNil() {
						t.Errorf("nil not expected for '%s' property", val.Type().Field(i).Name)
					}
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

func TestRoundTripper(t *testing.T) {
	mockTransport := &mock.MockRoundTripper{}
	client := mock.NewClient(false)
	config, _ := client.InClusterConfig()
	config.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
	semaphore := make(chan struct{}, 10)
	defer close(semaphore)
	// Create a new instance of the wrapped RoundTripper and pass in the mock RoundTripper.
	wrappedTransport := k8s.NewCustomTransport(config, semaphore, logging.New(io.Discard, 0, ""))
	wrappedTransport.SetBaseTransport(mockTransport)
	t.Run("successful_request", func(t *testing.T) {
		// Set up the mock to return a successful response.
		mockTransport.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		}
		// Make a request using the wrapped RoundTripper.
		_, err := wrappedTransport.RoundTrip(&http.Request{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("failed_request", func(t *testing.T) {
		// Set up the mock to return an error.
		mockTransport.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("Failed to send request")
		}

		// Make a request using the wrapped RoundTripper.
		_, err := wrappedTransport.RoundTrip(&http.Request{})
		if err == nil {
			t.Errorf("Expected an error but got none")
		}
	})
}
