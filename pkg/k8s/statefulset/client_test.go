package statefulset

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/mocks"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
	testTable := []struct {
		name            string
		statefulsetName string
		options         Options
		clientset       kubernetes.Interface
		errorMsg        error
	}{
		{
			name:            "create_statefulset",
			statefulsetName: "test_statefulset",
			clientset:       fake.NewSimpleClientset(),
			options: Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
			},
		},
		{
			name:            "update_statefulset",
			statefulsetName: "test_statefulset",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_statefulset",
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
			name:            "create_error",
			statefulsetName: "create_bad",
			clientset:       mocks.NewClientsetMock(),
			errorMsg:        fmt.Errorf("creating statefulset create_bad in namespace test: mock error: cannot create statefulset"),
		},
		{
			name:            "update_error",
			statefulsetName: "update_bad",
			clientset:       mocks.NewClientsetMock(),
			errorMsg:        fmt.Errorf("updating statefulset update_bad in namespace test: mock error: cannot update statefulset"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.statefulsetName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.statefulsetName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: test.options.Spec.ToK8S(),
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
		name            string
		statefulsetName string
		clientset       kubernetes.Interface
		errorMsg        error
	}{
		{
			name:            "delete_statefulset",
			statefulsetName: "test_statefulset",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset",
					Namespace: "test",
				},
			}),
		},
		{
			name:            "delete_not_found",
			statefulsetName: "test_statefulset_not_found",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset",
					Namespace: "test",
				},
			}),
		},
		{
			name:            "delete_error",
			statefulsetName: "delete_bad",
			clientset:       mocks.NewClientsetMock(),
			errorMsg:        fmt.Errorf("deleting statefulset delete_bad in namespace test: mock error: cannot delete statefulset"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			err := client.Delete(context.Background(), test.statefulsetName, "test")
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

func TestReadyReplicas(t *testing.T) {
	testTable := []struct {
		name            string
		statefulsetName string
		clientset       kubernetes.Interface
		expected        int32
		errorMsg        error
	}{
		{
			name:            "no_replicas_found",
			statefulsetName: "test_statefulset",
			clientset:       fake.NewSimpleClientset(),
			expected:        0,
		},
		{
			name:            "replicas_found",
			statefulsetName: "test_statefulset",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{ReadyReplicas: 5},
			}),
			expected: 5,
		},
		{
			name:            "replicas_error",
			statefulsetName: "statefulset_bad",
			clientset:       mocks.NewClientsetMock(),
			errorMsg:        fmt.Errorf("getting ReadyReplicas from statefulset statefulset_bad in namespace test: mock error: bad request"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			ready, err := client.ReadyReplicas(context.Background(), test.statefulsetName, "test")
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}

				if ready != test.expected {
					t.Errorf("response expected: %v, got: %v", test.expected, ready)
				}

			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if ready != 0 {
					t.Errorf("response not expected")
				}
			}
		})
	}
}
