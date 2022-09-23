package namespace_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper"
	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/namespace"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreate(t *testing.T) {
	testTable := []struct {
		name      string
		nsName    string
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:      "create_namespace",
			nsName:    "test",
			clientset: fake.NewSimpleClientset(),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := namespace.NewClient(test.clientset)
			response, err := client.Create(context.Background(), test.nsName)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: test.nsName,
						Annotations: map[string]string{
							"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
						},
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "beekeeper",
						},
					},
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

func TestUpdate(t *testing.T) {
	testTable := []struct {
		name      string
		nsName    string
		clientset kubernetes.Interface
		otpions   namespace.Options
		errorMsg  error
	}{
		{
			name:   "update_namespace",
			nsName: "test",
			clientset: fake.NewSimpleClientset(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
					},
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "beekeeper",
					},
				},
			}),
			otpions: namespace.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
			},
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := namespace.NewClient(test.clientset)
			response, err := client.Update(context.Background(), test.nsName, test.otpions)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.nsName,
						Annotations: test.otpions.Annotations,
						Labels:      test.otpions.Labels,
					},
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
		name      string
		nsName    string
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:   "delete_namespace",
			nsName: "test",
			clientset: fake.NewSimpleClientset(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
					},
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "beekeeper",
					},
				},
			}),
		},
		{
			name:      "no_namespaces",
			nsName:    "test",
			clientset: fake.NewSimpleClientset(),
			errorMsg:  fmt.Errorf("namespaces \"test\" not found"),
		},
		{
			name:   "not_managed_by_beekeeper",
			nsName: "test",
			clientset: fake.NewSimpleClientset(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
					},
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "not_beekeeper",
					},
				},
			}),
			errorMsg: fmt.Errorf("namespace test is not managed by beekeeper, try kubectl"),
		},
		{
			name:   "label_not_found",
			nsName: "test",
			clientset: fake.NewSimpleClientset(&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"created-by": fmt.Sprintf("beekeeper:%s", beekeeper.Version),
					},
					Labels: map[string]string{
						"label_not_found": "beekeeper",
					},
				},
			}),
			errorMsg: fmt.Errorf("namespace test is not managed by beekeeper, try kubectl"),
		},
		{
			name:      "delete_bad",
			nsName:    "test",
			clientset: mock.NewClientset(),
			errorMsg:  fmt.Errorf("deleting namespace test: mock error: namespace \"test\" can not be deleted"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := namespace.NewClient(test.clientset)
			err := client.Delete(context.Background(), test.nsName)
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
