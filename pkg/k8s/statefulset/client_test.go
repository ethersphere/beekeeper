package statefulset

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/ethersphere/beekeeper/mocks/k8s"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/autoscaling/v1"
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
			name:            "update_statefulset_on_delete",
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
				Spec:        StatefulSetSpec{UpdateStrategy: UpdateStrategy{Type: "OnDelete"}},
			},
		},
		{
			name:            "create_error",
			statefulsetName: "create_bad",
			clientset:       mock.NewClientset(),
			errorMsg:        fmt.Errorf("creating statefulset create_bad in namespace test: mock error: cannot create statefulset"),
		},
		{
			name:            "update_error",
			statefulsetName: "update_bad",
			clientset:       mock.NewClientset(),
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

				spec := test.options.Spec.ToK8S()

				if test.options.Spec.UpdateStrategy.Type == "OnDelete" {
					spec.UpdateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
					spec.UpdateStrategy.RollingUpdate = nil
				} else {
					spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
					spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
						Partition: &test.options.Spec.UpdateStrategy.RollingUpdatePartition,
					}
				}

				expected := &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.statefulsetName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: spec,
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
			clientset:       mock.NewClientset(),
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
			clientset:       mock.NewClientset(),
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

func TestRunningStatefulSets(t *testing.T) {
	testTable := []struct {
		name             string
		clientset        kubernetes.Interface
		namespace        string
		errorMsg         error
		expectedResponse []string
	}{
		{
			name:      "list_existing_sets",
			namespace: "test",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_1",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 1},
			}, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_2",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 1},
			}, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_3",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 2},
			}),
			expectedResponse: []string{"test_statefulset_1", "test_statefulset_2"},
		},
		{
			name:      "not_found_in_namespace",
			namespace: "test",
			clientset: mock.NewClientset(),
		},
		{
			name:      "wrong_namespace",
			namespace: "bad_test",
			clientset: mock.NewClientset(),
			errorMsg:  fmt.Errorf("list statefulsets in namespace bad_test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.RunningStatefulSets(context.Background(), test.namespace)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}

				if !reflect.DeepEqual(response, test.expectedResponse) {
					t.Errorf("response expected: %q, got: %q", response, test.expectedResponse)
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

func TestScale(t *testing.T) {
	testTable := []struct {
		name            string
		statefulSetName string
		clientset       kubernetes.Interface
		namespace       string
		errorMsg        error
	}{
		{
			name:            "scale_existing",
			namespace:       "test",
			statefulSetName: "test_statefulset_2",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_1",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 1},
			}, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_2",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 2},
			}),
		},
		{
			name:            "scale_not_existing_namespace",
			namespace:       "error_test",
			statefulSetName: "test_statefulset_2", clientset: fake.NewSimpleClientset(),
			errorMsg: fmt.Errorf("scaling statefulset test_statefulset_2 in namespace error_test: statefulsets.apps \"test_statefulset_2\" not found"),
		},
		{
			name:            "scale_not_existing_name",
			namespace:       "test",
			statefulSetName: "error_test",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_1",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 1},
			}),
			errorMsg: fmt.Errorf("scaling statefulset error_test in namespace test: statefulsets.apps \"error_test\" not found"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.Scale(context.Background(), test.statefulSetName, test.namespace, 3)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}

				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.Scale{
					ObjectMeta: metav1.ObjectMeta{
						Name:      test.statefulSetName,
						Namespace: test.namespace,
					},
					Spec: v1.ScaleSpec{
						Replicas: 3,
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

func TestStoppedStatefulSets(t *testing.T) {
	testTable := []struct {
		name             string
		clientset        kubernetes.Interface
		namespace        string
		errorMsg         error
		expectedResponse []string
	}{
		{
			name:      "list_existing_sets",
			namespace: "test",
			clientset: fake.NewSimpleClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_1",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 0},
			}, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_2",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 2},
			}, &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_statefulset_3",
					Namespace: "test",
				},
				Status: appsv1.StatefulSetStatus{Replicas: 0},
			}),
			expectedResponse: []string{"test_statefulset_1", "test_statefulset_3"},
		},
		{
			name:      "not_found_in_namespace",
			namespace: "test",
			clientset: mock.NewClientset(),
		},
		{
			name:      "wrong_namespace",
			namespace: "bad_test",
			clientset: mock.NewClientset(),
			errorMsg:  fmt.Errorf("list statefulsets in namespace bad_test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.StoppedStatefulSets(context.Background(), test.namespace)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}

				if !reflect.DeepEqual(response, test.expectedResponse) {
					t.Errorf("response expected: %q, got: %q", response, test.expectedResponse)
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
