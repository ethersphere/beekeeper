package service_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// newErrorClientset returns a fake clientset seeded with objects whose
// verb/resource action fails with err, used to exercise the error branches
// without a hand-written mock.
func newErrorClientset(verb, resource string, err error, objects ...runtime.Object) kubernetes.Interface {
	cs := fake.NewClientset(objects...)
	cs.PrependReactor(verb, resource, func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, err
	})
	return cs
}

func TestSet(t *testing.T) {
	t.Parallel()
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
			serviceName: "test_service",
			// No object seeded, so Get returns NotFound and Set falls through to
			// Create, which the reactor fails.
			clientset: newErrorClientset("create", "services", errors.New("mock error: cannot create service")),
			errorMsg:  fmt.Errorf("creating service test_service in namespace test: mock error: cannot create service"),
		},
		{
			name:        "update_error",
			serviceName: "test_service",
			// Seed the service so Get succeeds and Set reaches Update, which the
			// reactor fails.
			clientset: newErrorClientset("update", "services", errors.New("mock error: cannot update service"),
				&v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test_service", Namespace: "test"}}),
			errorMsg: fmt.Errorf("updating service test_service in namespace test: mock error: cannot update service"),
		},
		{
			name:        "get_error",
			serviceName: "test_service",
			// Get fails with a non-NotFound error, so Set returns the get error.
			clientset: newErrorClientset("get", "services", errors.New("mock error: unknown")),
			errorMsg:  fmt.Errorf("getting service test_service in namespace test: mock error: unknown"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := service.NewClient(test.clientset)
			response, err := client.Set(t.Context(), test.serviceName, "test", test.options)
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
	t.Parallel()
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
			serviceName: "test_service",
			clientset:   newErrorClientset("delete", "services", errors.New("mock error: cannot delete service")),
			errorMsg:    fmt.Errorf("deleting service test_service in namespace test: mock error: cannot delete service"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := service.NewClient(test.clientset)
			err := client.Delete(t.Context(), test.serviceName, "test")
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

// svc is a small helper for building a Service fixture in namespace "test".
func svc(name string, clusterIP string, labels map[string]string, ports ...v1.ServicePort) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			ClusterIP: clusterIP,
			Ports:     ports,
		},
	}
}

func TestGetNodes(t *testing.T) {
	t.Parallel()
	beeLabels := map[string]string{"node-group": "bee"}

	testTable := []struct {
		name          string
		labelSelector string
		clientset     kubernetes.Interface
		expected      []service.NodeInfo
		errorMsg      error
	}{
		{
			name:          "filters_by_api_port_and_cluster_ip",
			labelSelector: "node-group=bee",
			clientset: fake.NewClientset(
				// api port + real ClusterIP → included
				svc("bee-0", "10.0.0.1", beeLabels, v1.ServicePort{Name: "api", Port: 1633}),
				// api port but headless ClusterIP → excluded
				svc("bee-1", "None", beeLabels, v1.ServicePort{Name: "api", Port: 1633}),
				// matching labels but no api port → excluded
				svc("bee-2", "10.0.0.3", beeLabels, v1.ServicePort{Name: "p2p", Port: 1634}),
				// has api port but does not match the label selector → excluded
				svc("other-0", "10.0.0.4", map[string]string{"node-group": "other"}, v1.ServicePort{Name: "api", Port: 1633}),
			),
			expected: []service.NodeInfo{
				{Name: "bee-0", Endpoint: "http://10.0.0.1:1633"},
			},
		},
		{
			name:          "no_matching_services",
			labelSelector: "node-group=bee",
			clientset:     fake.NewClientset(),
			expected:      nil,
		},
		{
			name:          "list_error",
			labelSelector: "node-group=bee",
			clientset:     newErrorClientset("list", "services", errors.New("mock error")),
			errorMsg:      fmt.Errorf("listing services in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := service.NewClient(test.clientset)
			nodes, err := client.GetNodes(t.Context(), "test", test.labelSelector)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if !reflect.DeepEqual(nodes, test.expected) {
					t.Errorf("nodes expected: %#v, got: %#v", test.expected, nodes)
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if nodes != nil {
					t.Errorf("nodes not expected")
				}
			}
		})
	}
}

func TestFindNode(t *testing.T) {
	t.Parallel()
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "bee-0",
			Labels: map[string]string{"app": "bee", "node": "0"},
		},
	}

	testTable := []struct {
		name            string
		clientset       kubernetes.Interface
		expectedNode    *service.NodeInfo
		expectedSvcName string
		errorMsg        error
	}{
		{
			name: "match_found",
			clientset: fake.NewClientset(
				// nil selector → skipped
				svc("a-no-selector", "10.0.0.1", nil, v1.ServicePort{Name: "api", Port: 1633}),
				// selector does not match the pod labels → skipped
				selectorSvc("m-non-matching", "10.0.0.2", map[string]string{"app": "other"}, v1.ServicePort{Name: "api", Port: 1633}),
				// selector matches and has an api port → returned
				selectorSvc("z-matching-api", "10.0.0.9", map[string]string{"app": "bee"}, v1.ServicePort{Name: "api", Port: 1633}),
			),
			expectedNode:    &service.NodeInfo{Name: "z-matching-api", Endpoint: "http://10.0.0.9:1633"},
			expectedSvcName: "z-matching-api",
		},
		{
			name: "match_but_no_api_port",
			clientset: fake.NewClientset(
				// selector matches but no api port → falls through to the not-found error
				selectorSvc("matching-no-api", "10.0.0.5", map[string]string{"app": "bee"}, v1.ServicePort{Name: "p2p", Port: 1634}),
			),
			errorMsg: fmt.Errorf("no matching service found for pod bee-0"),
		},
		{
			name:      "list_error",
			clientset: newErrorClientset("list", "services", errors.New("mock error")),
			errorMsg:  fmt.Errorf("listing services in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := service.NewClient(test.clientset)
			node, svc, err := client.FindNode(t.Context(), "test", pod)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if !reflect.DeepEqual(node, test.expectedNode) {
					t.Errorf("node expected: %#v, got: %#v", test.expectedNode, node)
				}
				if svc == nil || svc.Name != test.expectedSvcName {
					t.Errorf("service expected with name %q, got: %#v", test.expectedSvcName, svc)
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if node != nil || svc != nil {
					t.Errorf("node/service not expected")
				}
			}
		})
	}
}

// selectorSvc builds a Service with a spec Selector (used by FindNode) in
// namespace "test".
func selectorSvc(name string, clusterIP string, selector map[string]string, ports ...v1.ServicePort) *v1.Service {
	s := svc(name, clusterIP, nil, ports...)
	s.Spec.Selector = selector
	return s
}
