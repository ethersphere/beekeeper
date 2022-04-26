package secret

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
	testTable := []struct {
		name       string
		secretName string
		options    Options
		clientset  kubernetes.Interface
		errorMsg   error
	}{
		{
			name:       "create_secret",
			secretName: "test_secret",
			clientset:  fake.NewSimpleClientset(),
			options: Options{
				Immutable:   true,
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
				Data:        map[string][]byte{"username": {1, 2, 3}, "password": {1, 2, 3}},
				StringData:  map[string]string{"username": "admin", "password": "t0p-Secret"},
				Type:        "Opaque",
			},
		},
		{
			name:       "update_secret",
			secretName: "test_secret",
			clientset: fake.NewSimpleClientset(&v1.Secret{
				StringData: map[string]string{"username": "a", "password": "t"},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_secret",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
				},
			}),
			options: Options{
				Immutable:   true,
				Annotations: map[string]string{"annotation_1": "annotation_value_updated"},
				Labels:      map[string]string{"label_1": "label_value_updated"},
				Data:        map[string][]byte{"username": {1, 2, 3}, "password": {1, 2, 3}},
				StringData:  map[string]string{"username": "admin", "password": "t0p-Secret"},
				Type:        "Opaque",
			},
		},
		{
			name:       "create_error",
			secretName: "create_bad",
			clientset:  mocks.NewClientsetMock(),
			errorMsg:   fmt.Errorf("creating secret create_bad in namespace test: mock error: cannot create secret"),
		},
		{
			name:       "update_error",
			secretName: "update_bad",
			clientset:  mocks.NewClientsetMock(),
			errorMsg:   fmt.Errorf("updating secret update_bad in namespace test: mock error: cannot update secret"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.secretName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.secretName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Immutable:  &test.options.Immutable,
					Data:       test.options.Data,
					StringData: test.options.StringData,
					Type:       v1.SecretType(test.options.Type),
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
		name       string
		secretName string
		clientset  kubernetes.Interface
		errorMsg   error
	}{
		{
			name:       "delete_secret",
			secretName: "test_secret",
			clientset: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_secret",
					Namespace: "test",
				},
			}),
		},
		{
			name:       "delete_not_found",
			secretName: "test_secret_not_found",
			clientset: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_secret",
					Namespace: "test",
				},
			}),
		},
		{
			name:       "delete_error",
			secretName: "delete_bad",
			clientset:  mocks.NewClientsetMock(),
			errorMsg:   fmt.Errorf("deleting secret delete_bad in namespace test: mock error: cannot delete secret"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			err := client.Delete(context.Background(), test.secretName, "test")
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
