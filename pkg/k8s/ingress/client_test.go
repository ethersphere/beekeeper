package ingress

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
	testTable := []struct {
		name        string
		ingressName string
		options     Options
		clientset   kubernetes.Interface
		errorMsg    error
	}{
		{
			name:        "create_ingress",
			ingressName: "test_ingress",
			clientset:   fake.NewSimpleClientset(),
			options: Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
				Spec: Spec{
					Rules: []Rule{
						{
							Host: "host",
							Paths: []Path{
								{
									Backend:  Backend{ServiceName: "sc_name", ServicePort: "9999"},
									Path:     "/test",
									PathType: "absolute",
								},
							},
						},
					},
					TLS: []TLS{
						{
							Hosts:      []string{"host1", "host2"},
							SecretName: "secret",
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
			options: Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
		},
		{
			name:        "create_error",
			ingressName: "create_bad",
			clientset:   mocks.NewClientset(),
			errorMsg:    fmt.Errorf("creating ingress create_bad in namespace test: mock error: cannot create ingress"),
		},
		{
			name:        "update_error",
			ingressName: "update_bad",
			clientset:   mocks.NewClientset(),
			errorMsg:    fmt.Errorf("updating ingress update_bad in namespace test: mock error: cannot update ingress"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
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
					Spec: test.options.Spec.toK8S(),
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
			clientset:   mocks.NewClientset(),
			errorMsg:    fmt.Errorf("deleting ingress delete_bad in namespace test: mock error: cannot delete ingress"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
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
