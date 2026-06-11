package statefulset_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/logging"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
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

// newStatefulSet builds a StatefulSet fixture in namespace "test" with a single
// container, so UpdateImage (which indexes Containers[0]) can run against it.
func newStatefulSet(name, image string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "bee", Image: image}},
				},
			},
		},
	}
}

func TestSet(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		name            string
		statefulsetName string
		options         statefulset.Options
		clientset       kubernetes.Interface
		errorMsg        error
	}{
		{
			name:            "create_statefulset",
			statefulsetName: "test_statefulset",
			clientset:       fake.NewSimpleClientset(),
			options: statefulset.Options{
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
			options: statefulset.Options{
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
			options: statefulset.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
				Spec:        statefulset.StatefulSetSpec{UpdateStrategy: statefulset.UpdateStrategy{Type: "OnDelete"}},
			},
		},
		{
			name:            "create_error",
			statefulsetName: "test_statefulset",
			// No object seeded, so Update returns NotFound and Set falls through
			// to Create, which the reactor fails.
			clientset: newErrorClientset("create", "statefulsets", errors.New("mock error: cannot create statefulset")),
			errorMsg:  fmt.Errorf("creating statefulset test_statefulset in namespace test: mock error: cannot create statefulset"),
		},
		{
			name:            "update_error",
			statefulsetName: "test_statefulset",
			clientset:       newErrorClientset("update", "statefulsets", errors.New("mock error: cannot update statefulset")),
			errorMsg:        fmt.Errorf("updating statefulset test_statefulset in namespace test: mock error: cannot update statefulset"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			response, err := client.Set(t.Context(), test.statefulsetName, "test", test.options)
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
						Partition: test.options.Spec.UpdateStrategy.RollingUpdatePartition,
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
	t.Parallel()
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
			statefulsetName: "test_statefulset",
			clientset:       newErrorClientset("delete", "statefulsets", errors.New("mock error: cannot delete statefulset")),
			errorMsg:        fmt.Errorf("deleting statefulset test_statefulset in namespace test: mock error: cannot delete statefulset"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			err := client.Delete(t.Context(), test.statefulsetName, "test")
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
	t.Parallel()
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
			statefulsetName: "test_statefulset",
			// Get fails with a non-NotFound error, so ReadyReplicas surfaces it.
			clientset: newErrorClientset("get", "statefulsets", errors.New("mock error: bad request")),
			errorMsg:  fmt.Errorf("getting ReadyReplicas from statefulset test_statefulset in namespace test: mock error: bad request"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			ready, err := client.ReadyReplicas(t.Context(), test.statefulsetName, "test")
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

// watchClientset returns a fake clientset whose StatefulSet Watch is served by
// w (and fails with watchErr if non-nil), per decisions §3.
func watchClientset(w *watch.RaceFreeFakeWatcher, watchErr error) kubernetes.Interface {
	cs := fake.NewClientset()
	cs.PrependWatchReactor("statefulsets", k8stesting.DefaultWatchReactor(w, watchErr))
	return cs
}

// readySts is a StatefulSet event payload with the given replica counts.
func readySts(replicas, ready int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{Status: appsv1.StatefulSetStatus{Replicas: replicas, ReadyReplicas: ready}}
}

func TestReadyReplicasWatch(t *testing.T) {
	t.Parallel()
	t.Run("watch_error", func(t *testing.T) {
		cs := watchClientset(nil, errors.New("mock error: bad request"))
		client := statefulset.NewClient(cs, logging.New(io.Discard, 0))
		_, err := client.ReadyReplicasWatch(t.Context(), "test_statefulset", "test")
		if err == nil || err.Error() != "getting ready from statefulset test_statefulset in namespace test: mock error: bad request" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ready", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		// RaceFreeFake buffers events, so enqueue before the watch starts.
		w.Modify(readySts(1, 1))
		client := statefulset.NewClient(watchClientset(w, nil), logging.New(io.Discard, 0))
		ready, err := client.ReadyReplicasWatch(t.Context(), "test_statefulset", "test")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if ready != 1 {
			t.Errorf("ready expected: 1, got: %d", ready)
		}
	})

	t.Run("not_ready_then_ready", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		w.Modify(readySts(2, 1)) // replicas != readyReplicas → loop continues
		w.Modify(readySts(2, 2)) // matches → returns
		client := statefulset.NewClient(watchClientset(w, nil), logging.New(io.Discard, 0))
		ready, err := client.ReadyReplicasWatch(t.Context(), "test_statefulset", "test")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if ready != 2 {
			t.Errorf("ready expected: 2, got: %d", ready)
		}
	})

	t.Run("non_statefulset_event_then_ready", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		w.Add(&corev1.Pod{}) // not a *StatefulSet → type assertion fails, loop continues
		w.Modify(readySts(1, 1))
		client := statefulset.NewClient(watchClientset(w, nil), logging.New(io.Discard, 0))
		ready, err := client.ReadyReplicasWatch(t.Context(), "test_statefulset", "test")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if ready != 1 {
			t.Errorf("ready expected: 1, got: %d", ready)
		}
	})

	t.Run("channel_closed", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		w.Stop() // closes the result channel
		client := statefulset.NewClient(watchClientset(w, nil), logging.New(io.Discard, 0))
		_, err := client.ReadyReplicasWatch(t.Context(), "test_statefulset", "test")
		if err == nil || err.Error() != "watch channel closed" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("context_cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		w := watch.NewRaceFreeFake()
		client := statefulset.NewClient(watchClientset(w, nil), logging.New(io.Discard, 0))
		_, err := client.ReadyReplicasWatch(ctx, "test_statefulset", "test")
		if err == nil || err.Error() != "context canceled" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ticker_logs_until_timeout", func(t *testing.T) {
		// Shorten the "not ready yet" log cadence so the ticker branch fires
		// well before the context deadline; no event is ever sent.
		restore := statefulset.SetNotReadyLogInterval(time.Millisecond)
		defer restore()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		w := watch.NewRaceFreeFake()
		client := statefulset.NewClient(watchClientset(w, nil), logging.New(io.Discard, 0))
		_, err := client.ReadyReplicasWatch(ctx, "test_statefulset", "test")
		if err == nil || err.Error() != "context deadline exceeded" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestRunningStatefulSets(t *testing.T) {
	t.Parallel()
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
			// List returns NotFound, which RunningStatefulSets treats as empty.
			clientset: newErrorClientset("list", "statefulsets", apierrors.NewNotFound(schema.GroupResource{}, "test")),
		},
		{
			name:      "wrong_namespace",
			namespace: "bad_test",
			clientset: newErrorClientset("list", "statefulsets", errors.New("mock error")),
			errorMsg:  fmt.Errorf("list statefulsets in namespace bad_test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			response, err := client.RunningStatefulSets(t.Context(), test.namespace)
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
	t.Parallel()
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
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			response, err := client.Scale(t.Context(), test.statefulSetName, test.namespace, 3)
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
	t.Parallel()
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
			// List returns NotFound, which StoppedStatefulSets treats as empty.
			clientset: newErrorClientset("list", "statefulsets", apierrors.NewNotFound(schema.GroupResource{}, "test")),
		},
		{
			name:      "wrong_namespace",
			namespace: "bad_test",
			clientset: newErrorClientset("list", "statefulsets", errors.New("mock error")),
			errorMsg:  fmt.Errorf("list statefulsets in namespace bad_test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			response, err := client.StoppedStatefulSets(t.Context(), test.namespace)
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

func TestStatefulSets(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		name          string
		labelSelector string
		clientset     kubernetes.Interface
		expectedNames []string
		errorMsg      error
	}{
		{
			name:          "list_existing",
			labelSelector: "app=bee",
			clientset: fake.NewClientset(
				&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "sts-0", Namespace: "test", Labels: map[string]string{"app": "bee"}}},
				&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "sts-1", Namespace: "test", Labels: map[string]string{"app": "bee"}}},
				&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "test", Labels: map[string]string{"app": "other"}}},
			),
			expectedNames: []string{"sts-0", "sts-1"},
		},
		{
			name:          "list_error",
			labelSelector: "app=bee",
			clientset:     newErrorClientset("list", "statefulsets", errors.New("mock error")),
			errorMsg:      fmt.Errorf("list statefulsets in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			list, err := client.StatefulSets(t.Context(), "test", test.labelSelector)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				names := make([]string, 0, len(list))
				for _, s := range list {
					names = append(names, s.Name)
				}
				if !reflect.DeepEqual(names, test.expectedNames) {
					t.Errorf("names expected: %q, got: %q", test.expectedNames, names)
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if list != nil {
					t.Errorf("list not expected")
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		name      string
		stsName   string
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:      "get_existing",
			stsName:   "sts-0",
			clientset: fake.NewClientset(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "sts-0", Namespace: "test"}}),
		},
		{
			name:      "get_error",
			stsName:   "sts-0",
			clientset: newErrorClientset("get", "statefulsets", errors.New("mock error")),
			errorMsg:  fmt.Errorf("getting statefulset sts-0 in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			sts, err := client.Get(t.Context(), test.stsName, "test")
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if sts == nil || sts.Name != test.stsName {
					t.Errorf("statefulset expected with name %q, got: %#v", test.stsName, sts)
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if sts != nil {
					t.Errorf("statefulset not expected")
				}
			}
		})
	}
}

func TestUpdateImage(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		cs := fake.NewClientset(newStatefulSet("sts-0", "bee:old"))
		client := statefulset.NewClient(cs, logging.New(io.Discard, 0))
		if err := client.UpdateImage(t.Context(), "sts-0", "test", "bee:new"); err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		got, err := cs.AppsV1().StatefulSets("test").Get(t.Context(), "sts-0", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("getting updated statefulset: %s", err.Error())
		}
		if img := got.Spec.Template.Spec.Containers[0].Image; img != "bee:new" {
			t.Errorf("image expected: %q, got: %q", "bee:new", img)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		client := statefulset.NewClient(fake.NewClientset(), logging.New(io.Discard, 0))
		if err := client.UpdateImage(t.Context(), "sts-0", "test", "bee:new"); err != nil {
			t.Errorf("error not expected for missing statefulset, got: %s", err.Error())
		}
	})

	t.Run("get_error", func(t *testing.T) {
		client := statefulset.NewClient(newErrorClientset("get", "statefulsets", errors.New("mock error")), logging.New(io.Discard, 0))
		err := client.UpdateImage(t.Context(), "sts-0", "test", "bee:new")
		if err == nil || err.Error() != "getting statefulset sts-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("update_error", func(t *testing.T) {
		cs := newErrorClientset("update", "statefulsets", errors.New("mock error"), newStatefulSet("sts-0", "bee:old"))
		client := statefulset.NewClient(cs, logging.New(io.Discard, 0))
		err := client.UpdateImage(t.Context(), "sts-0", "test", "bee:new")
		if err == nil || err.Error() != "updating statefulset sts-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestGetUpdateStrategy(t *testing.T) {
	t.Parallel()
	partition := int32(2)

	testTable := []struct {
		name              string
		clientset         kubernetes.Interface
		expectedType      string
		expectedPartition *int32
		errorMsg          error
	}{
		{
			name: "on_delete",
			clientset: fake.NewClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "sts-0", Namespace: "test"},
				Spec:       appsv1.StatefulSetSpec{UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType}},
			}),
			expectedType: statefulset.UpdateStrategyOnDelete,
		},
		{
			name: "rolling_with_partition",
			clientset: fake.NewClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "sts-0", Namespace: "test"},
				Spec: appsv1.StatefulSetSpec{UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
					Type:          appsv1.RollingUpdateStatefulSetStrategyType,
					RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: &partition},
				}},
			}),
			expectedType:      statefulset.UpdateStrategyRolling,
			expectedPartition: &partition,
		},
		{
			name: "rolling_without_partition",
			clientset: fake.NewClientset(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "sts-0", Namespace: "test"},
				Spec:       appsv1.StatefulSetSpec{UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.RollingUpdateStatefulSetStrategyType}},
			}),
			expectedType: statefulset.UpdateStrategyRolling,
		},
		{
			name:         "not_found",
			clientset:    fake.NewClientset(),
			expectedType: "",
		},
		{
			name:      "get_error",
			clientset: newErrorClientset("get", "statefulsets", errors.New("mock error")),
			errorMsg:  fmt.Errorf("getting statefulset sts-0 in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			strategy, err := client.GetUpdateStrategy(t.Context(), "sts-0", "test")
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if strategy.Type != test.expectedType {
					t.Errorf("type expected: %q, got: %q", test.expectedType, strategy.Type)
				}
				switch {
				case test.expectedPartition == nil && strategy.RollingUpdatePartition != nil:
					t.Errorf("partition expected nil, got: %d", *strategy.RollingUpdatePartition)
				case test.expectedPartition != nil && (strategy.RollingUpdatePartition == nil || *strategy.RollingUpdatePartition != *test.expectedPartition):
					t.Errorf("partition expected: %d, got: %v", *test.expectedPartition, strategy.RollingUpdatePartition)
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

func TestUpdate(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		name        string
		statefulSet *appsv1.StatefulSet
		clientset   kubernetes.Interface
		errorMsg    error
	}{
		{
			name:        "nil_statefulset",
			statefulSet: nil,
			clientset:   fake.NewClientset(),
			errorMsg:    fmt.Errorf("statefulSet cannot be nil"),
		},
		{
			name:        "update_success",
			statefulSet: newStatefulSet("sts-0", "bee:new"),
			clientset:   fake.NewClientset(newStatefulSet("sts-0", "bee:old")),
		},
		{
			name:        "update_error",
			statefulSet: newStatefulSet("sts-0", "bee:new"),
			clientset:   newErrorClientset("update", "statefulsets", errors.New("mock error")),
			errorMsg:    fmt.Errorf("updating statefulset sts-0 in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := statefulset.NewClient(test.clientset, logging.New(io.Discard, 0))
			err := client.Update(t.Context(), "test", test.statefulSet)
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
