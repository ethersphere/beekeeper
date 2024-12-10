package service_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
	testTable := []struct {
		name        string
		serviceName string
		options     service.Options
		clientset   kubernetes.Interface
		errorMsg    error
	}{
		{
			name:        "create_service",
			serviceName: "test_service",
			clientset:   fake.NewSimpleClientset(),
			options: service.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
			},
		},
		{
			name:        "update_service",
			serviceName: "test_service",
			clientset: fake.NewSimpleClientset(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_service",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
				},
			}),
			options: service.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
		},
		{
			name:        "spec_ports",
			serviceName: "test_service",
			clientset:   fake.NewSimpleClientset(),
			options: service.Options{
				ServiceSpec: service.Spec{
					Ports: []service.Port{
						{
							Name: "http8080", Port: 8080, AppProtocol: "http", Nodeport: 80, Protocol: "http", TargetPort: "80",
						},
						{
							Name: "https8081", Port: 8081, AppProtocol: "https", Nodeport: 443, Protocol: "https", TargetPort: "443",
						},
					},
				},
			},
		},
		{
			name:        "create_error",
			serviceName: mock.CreateBad,
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("creating service create_bad in namespace test: mock error: cannot create service"),
		},
		{
			name:        "update_error",
			serviceName: mock.UpdateBad,
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("updating service update_bad in namespace test: mock error: cannot update service"),
		},
		{
			name:        "get_error",
			serviceName: "get_bad",
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("getting service get_bad in namespace test: mock error: unknown"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := service.NewClient(test.clientset, "cluster.local")
			response, err := client.Set(context.Background(), test.serviceName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.serviceName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: test.options.ServiceSpec.ToK8S(),
				}

				if !reflect.DeepEqual(response, expected) {
					t.Errorf("response expected: %q, got: %q", response, expected)
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if response != nil {
					t.Errorf("response not expected")
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testTable := []struct {
		name        string
		serviceName string
		clientset   kubernetes.Interface
		errorMsg    error
	}{
		{
			name:        "delete_service",
			serviceName: "test_service",
			clientset: fake.NewSimpleClientset(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_service",
					Namespace: "test",
				},
			}),
		},
		{
			name:        "delete_not_found",
			serviceName: "test_service_not_found",
			clientset: fake.NewSimpleClientset(&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_service",
					Namespace: "test",
				},
			}),
		},
		{
			name:        "delete_error",
			serviceName: mock.DeleteBad,
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("deleting service delete_bad in namespace test: mock error: cannot delete service"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := service.NewClient(test.clientset, "cluster.local")
			err := client.Delete(context.Background(), test.serviceName, "test")
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
			}
		})
	}
}
