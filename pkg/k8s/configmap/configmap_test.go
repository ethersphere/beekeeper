package configmap_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
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
		name       string
		configName string
		options    configmap.Options
		clientset  kubernetes.Interface
		errorMsg   error
	}{
		{
			name:       "create_config_map",
			configName: "test_config_map",
			clientset:  fake.NewSimpleClientset(),
			options: configmap.Options{
				Immutable:   true,
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
				Data:        map[string]string{"data_1": "data_value_1"},
				BinaryData:  map[string][]byte{"bd_1": {1, 2, 3}},
			},
		},
		{
			name:       "update_config_map",
			configName: "test_config_map",
			clientset: fake.NewSimpleClientset(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_config_map",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
				},
				BinaryData: map[string][]byte{"bd_1": {1, 2, 3}},
				Data:       map[string]string{"data_1": "data_value_1"},
			}),
			options: configmap.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
		},
		{
			name:       "create_error",
			configName: "test_config_map",
			// No object seeded, so Update returns NotFound and Set falls through
			// to Create, which the reactor fails.
			clientset: newErrorClientset("create", "configmaps", errors.New("mock error: cannot create config map")),
			errorMsg:  fmt.Errorf("creating configmap test_config_map in namespace test: mock error: cannot create config map"),
		},
		{
			name:       "update_error",
			configName: "test_config_map",
			clientset:  newErrorClientset("update", "configmaps", errors.New("mock error: cannot update config map")),
			errorMsg:   fmt.Errorf("updating configmap test_config_map in namespace test: mock error: cannot update config map"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := configmap.NewClient(test.clientset)
			response, err := client.Set(t.Context(), test.configName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.configName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Immutable:  &test.options.Immutable,
					BinaryData: test.options.BinaryData,
					Data:       test.options.Data,
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
		name       string
		configName string
		clientset  kubernetes.Interface
		errorMsg   error
	}{
		{
			name:       "delete_config_map",
			configName: "test_config_map",
			clientset: fake.NewSimpleClientset(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_config_map",
					Namespace: "test",
				},
			}),
		},
		{
			name:       "delete_not_found",
			configName: "test_config_map_not_found",
			clientset: fake.NewSimpleClientset(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_config_map",
					Namespace: "test",
				},
			}),
		},
		{
			name:       "delete_error",
			configName: "test_config_map",
			clientset:  newErrorClientset("delete", "configmaps", errors.New("mock error: cannot delete config map")),
			errorMsg:   fmt.Errorf("deleting configmap test_config_map in namespace test: mock error: cannot delete config map"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := configmap.NewClient(test.clientset)
			err := client.Delete(t.Context(), test.configName, "test")
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
