package pod

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
		name      string
		podName   string
		options   Options
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:      "create_pod",
			podName:   "test_pod",
			clientset: fake.NewSimpleClientset(),
			options: Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
			},
		},
		{
			name:    "update_pod",
			podName: "test_pod",
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_pod",
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
			name:      "create_error",
			podName:   "create_bad",
			clientset: mocks.NewClientsetMock(),
			errorMsg:  fmt.Errorf("creating pod create_bad in namespace test: mock error: cannot create pod"),
		},
		{
			name:      "update_error",
			podName:   "update_bad",
			clientset: mocks.NewClientsetMock(),
			errorMsg:  fmt.Errorf("updating pod update_bad in namespace test: mock error: cannot update pod"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.podName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.podName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: test.options.PodSpec.toK8S(),
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
		podName   string
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:    "delete_pod",
			podName: "test_pod",
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_pod",
					Namespace: "test",
				},
			}),
		},
		{
			name:    "delete_not_found",
			podName: "test_pod_not_found",
			clientset: fake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_pod",
					Namespace: "test",
				},
			}),
		},
		{
			name:      "delete_error",
			podName:   "delete_bad",
			clientset: mocks.NewClientsetMock(),
			errorMsg:  fmt.Errorf("deleting pod delete_bad in namespace test: mock error: cannot delete pod"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			err := client.Delete(context.Background(), test.podName, "test")
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