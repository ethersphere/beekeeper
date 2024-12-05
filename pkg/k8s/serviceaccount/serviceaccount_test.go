package serviceaccount_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
	testTable := []struct {
		name       string
		secretName string
		options    serviceaccount.Options
		clientset  kubernetes.Interface
		errorMsg   error
	}{
		{
			name:       "create_service_account",
			secretName: "test_service_account",
			clientset:  fake.NewSimpleClientset(),
			options: serviceaccount.Options{
				Annotations:                  map[string]string{"annotation_1": "annotation_value_1"},
				Labels:                       map[string]string{"label_1": "label_value_1"},
				AutomountServiceAccountToken: true,
				ImagePullSecrets:             []string{"image_secret_1", "image_secret_2"},
				Secrets:                      []string{"secret_1", "secret_2"},
			},
		},
		{
			name:       "update_service_account",
			secretName: "test_service_account",
			clientset: fake.NewSimpleClientset(&v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_service_account",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
				},
			}),
			options: serviceaccount.Options{
				Annotations:                  map[string]string{"annotation_1": "annotation_value_updated"},
				Labels:                       map[string]string{"label_1": "label_value_updated"},
				AutomountServiceAccountToken: true,
				ImagePullSecrets:             []string{"image_secret_1", "image_secret_2"},
				Secrets:                      []string{"secret_1", "secret_2"},
			},
		},
		{
			name:       "create_error",
			secretName: mock.CreateBad,
			clientset:  mock.NewClientset(),
			errorMsg:   fmt.Errorf("creating service account create_bad in namespace test: mock error: cannot create service account"),
		},
		{
			name:       "update_error",
			secretName: mock.UpdateBad,
			clientset:  mock.NewClientset(),
			errorMsg:   fmt.Errorf("updating service account update_bad in namespace test: mock error: cannot update service account"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := serviceaccount.NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.secretName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.secretName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					AutomountServiceAccountToken: &test.options.AutomountServiceAccountToken,
					ImagePullSecrets: func() (l []v1.LocalObjectReference) {
						for _, s := range test.options.ImagePullSecrets {
							l = append(l, v1.LocalObjectReference{Name: s})
						}
						return
					}(),
					Secrets: func() (l []v1.ObjectReference) {
						for _, s := range test.options.Secrets {
							l = append(l, v1.ObjectReference{Name: s})
						}
						return
					}(),
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
			name:       "delete_service_account",
			secretName: "test_service_account",
			clientset: fake.NewSimpleClientset(&v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_service_account",
					Namespace: "test",
				},
			}),
		},
		{
			name:       "delete_not_found",
			secretName: "test_service_account_not_found",
			clientset: fake.NewSimpleClientset(&v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_service_account",
					Namespace: "test",
				},
			}),
		},
		{
			name:       "delete_error",
			secretName: mock.DeleteBad,
			clientset:  mock.NewClientset(),
			errorMsg:   fmt.Errorf("deleting service account delete_bad in namespace test: mock error: cannot delete service account"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := serviceaccount.NewClient(test.clientset)
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
