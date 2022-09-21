package ingress_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
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
					Backend: ingress.Backend{
						ServiceName: "service_name",
						ServicePort: "service_port",
					},
					Rules: []ingress.Rule{
						{
							Host: "host",
							Paths: []ingress.Path{
								{
									Backend:  ingress.Backend{ServiceName: "sc_name", ServicePort: "9999"},
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
			ingressName: "create_bad",
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("creating ingress create_bad in namespace test: mock error: cannot create ingress"),
		},
		{
			name:        "update_error",
			ingressName: "update_bad",
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("updating ingress update_bad in namespace test: mock error: cannot update ingress"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := ingress.NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.ingressName, "test", test.options)
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
			ingressName: "delete_bad",
			clientset:   mock.NewClientset(),
			errorMsg:    fmt.Errorf("deleting ingress delete_bad in namespace test: mock error: cannot delete ingress"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := ingress.NewClient(test.clientset)
			err := client.Delete(context.Background(), test.ingressName, "test")
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
