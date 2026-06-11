package ingress_test

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/logging"
	v1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		name         string
		ingressName  string
		options      ingress.Options
		clientset    kubernetes.Interface
		expectedSpec v1.IngressSpec
		errorMsg     error
	}{
		{
			name:        "create_ingress",
			ingressName: "test_ingress",
			clientset:   fake.NewSimpleClientset(),
			options: ingress.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
				Spec: ingress.Spec{
					Class: "class",
					Rules: []ingress.Rule{
						{
							Host: "host",
							Paths: []ingress.Path{
								{
									Backend:  ingress.Backend{ServiceName: "sc_name", ServicePortName: "9999"},
									Path:     "/test",
									PathType: "absolute",
								},
							},
						},
					},
					TLS: []ingress.TLS{
						{
							Hosts:      []string{"host1", "host2"},
							SecretName: "secret",
						},
					},
				},
			},
			expectedSpec: v1.IngressSpec{
				IngressClassName: func() *string {
					class := "class"
					return &class
				}(),
				TLS: []v1.IngressTLS{
					{
						Hosts:      []string{"host1", "host2"},
						SecretName: "secret",
					},
				},
				Rules: []v1.IngressRule{
					{
						Host: "host",
						IngressRuleValue: v1.IngressRuleValue{
							HTTP: &v1.HTTPIngressRuleValue{
								Paths: []v1.HTTPIngressPath{
									{
										Backend: v1.IngressBackend{
											Service: &v1.IngressServiceBackend{
												Name: "sc_name",
												Port: v1.ServiceBackendPort{
													Name: "9999",
												},
											},
										},
										Path: "/test",
										PathType: func() *v1.PathType {
											pt := v1.PathType("absolute")
											return &pt
										}(),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:        "update_ingress",
			ingressName: "test_ingress",
			clientset: fake.NewSimpleClientset(&v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_ingress",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
				},
			}),
			options: ingress.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
			expectedSpec: v1.IngressSpec{},
		},
		{
			name:        "create_error",
			ingressName: "test_ingress",
			// No object seeded, so Update returns NotFound and Set falls through
			// to Create, which the reactor fails.
			clientset: newErrorClientset("create", "ingresses", errors.New("mock error: cannot create ingress")),
			errorMsg:  fmt.Errorf("creating ingress test_ingress in namespace test: mock error: cannot create ingress"),
		},
		{
			name:        "update_error",
			ingressName: "test_ingress",
			clientset:   newErrorClientset("update", "ingresses", errors.New("mock error: cannot update ingress")),
			errorMsg:    fmt.Errorf("updating ingress test_ingress in namespace test: mock error: cannot update ingress"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := ingress.NewClient(test.clientset, logging.New(io.Discard, 0))
			response, err := client.Set(t.Context(), test.ingressName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.ingressName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: test.expectedSpec,
				}
				if !reflect.DeepEqual(*response, *expected) {
					t.Errorf("response expected: %#v, got: %#v", expected, response)
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
		ingressName string
		clientset   kubernetes.Interface
		errorMsg    error
	}{
		{
			name:        "delete_ingress",
			ingressName: "test_ingress",
			clientset: fake.NewSimpleClientset(&v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_ingress",
					Namespace: "test",
				},
			}),
		},
		{
			name:        "delete_not_found",
			ingressName: "test_ingress_not_found",
			clientset: fake.NewSimpleClientset(&v1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_ingress",
					Namespace: "test",
				},
			}),
		},
		{
			name:        "delete_error",
			ingressName: "test_ingress",
			clientset:   newErrorClientset("delete", "ingresses", errors.New("mock error: cannot delete ingress")),
			errorMsg:    fmt.Errorf("deleting ingress test_ingress in namespace test: mock error: cannot delete ingress"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := ingress.NewClient(test.clientset, logging.New(io.Discard, 0))
			err := client.Delete(t.Context(), test.ingressName, "test")
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

// func TestToK8s(t *testing.T) {
// 	testTable := []struct {
// 		name     string
// 		options  ingress.Options
// 		errorMsg error
// 	}{
// 		{
// 			name:     "test",
// 			options:  ingress.Options{},
// 			errorMsg: nil,
// 		},
// 	}

// 	for _, test := range testTable {
// 		t.Run(test.name, func(t *testing.T) {
// 			ingresSpec := test.options.Spec.ToK8S()
// 			if len(ingresSpec.Rules) == 0 {
// 			}
// 		})
// 	}
// }

// ingressWith builds an Ingress in namespace "test" with one rule per host.
func ingressWith(name string, labels map[string]string, hosts ...string) *v1.Ingress {
	ing := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test", Labels: labels},
	}
	for _, h := range hosts {
		ing.Spec.Rules = append(ing.Spec.Rules, v1.IngressRule{Host: h})
	}
	return ing
}

func TestGetNodes(t *testing.T) {
	t.Parallel()
	beeLabels := map[string]string{"app": "bee"}

	testTable := []struct {
		name      string
		label     string
		clientset kubernetes.Interface
		expected  []ingress.NodeInfo
		errorMsg  error
	}{
		{
			name:  "lists_hosts_by_label",
			label: "app=bee",
			clientset: fake.NewClientset(
				// two rules, one with an empty host (skipped)
				ingressWith("ing-0", beeLabels, "bee0.example.com", ""),
				ingressWith("ing-1", beeLabels, "bee1.example.com"),
				// excluded by the label selector
				ingressWith("other", map[string]string{"app": "other"}, "other.example.com"),
			),
			expected: []ingress.NodeInfo{
				{Name: "ing-0", Host: "bee0.example.com"},
				{Name: "ing-1", Host: "bee1.example.com"},
			},
		},
		{
			name:      "no_matching_ingresses",
			label:     "app=bee",
			clientset: fake.NewClientset(),
			expected:  nil,
		},
		{
			name:      "not_found",
			label:     "app=bee",
			clientset: newErrorClientset("list", "ingresses", apierrors.NewNotFound(schema.GroupResource{}, "test")),
			expected:  nil,
		},
		{
			name:      "list_error",
			label:     "app=bee",
			clientset: newErrorClientset("list", "ingresses", errors.New("mock error")),
			errorMsg:  fmt.Errorf("list ingresses in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := ingress.NewClient(test.clientset, logging.New(io.Discard, 0))
			nodes, err := client.GetNodes(t.Context(), "test", test.label)
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
