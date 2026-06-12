package pod_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/ethersphere/beekeeper/pkg/k8s/internal/k8stest"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/logging"
)

func TestSet(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		name      string
		podName   string
		options   pod.Options
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:      "create_pod",
			podName:   "test_pod",
			clientset: fake.NewSimpleClientset(),
			options: pod.Options{
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
			options: pod.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
		},
		{
			name:    "create_error",
			podName: "test_pod",
			// No object seeded, so Update returns NotFound and Set falls through
			// to Create, which the reactor fails.
			clientset: k8stest.NewErrorClientset("create", "pods", errors.New("mock error: cannot create pod")),
			errorMsg:  fmt.Errorf("creating pod test_pod in namespace test: mock error: cannot create pod"),
		},
		{
			name:      "update_error",
			podName:   "test_pod",
			clientset: k8stest.NewErrorClientset("update", "pods", errors.New("mock error: cannot update pod")),
			errorMsg:  fmt.Errorf("updating pod test_pod in namespace test: mock error: cannot update pod"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pod.NewClient(test.clientset, logging.New(io.Discard, 0))
			response, err := client.Set(t.Context(), test.podName, "test", test.options)
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
					Spec: newDefaultPodTemplateSpec().Spec,
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
			podName:   "test_pod",
			clientset: k8stest.NewErrorClientset("delete", "pods", errors.New("mock error: cannot delete pod")),
			errorMsg:  fmt.Errorf("deleting pod test_pod in namespace test: mock error: cannot delete pod"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pod.NewClient(test.clientset, logging.New(io.Discard, 0))
			_, err := client.Delete(t.Context(), test.podName, "test")
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

// newPod builds a Pod fixture in namespace "test".
func newPod(name string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test", Labels: labels},
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		name      string
		podName   string
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:      "get_existing",
			podName:   "p0",
			clientset: fake.NewClientset(newPod("p0", nil)),
		},
		{
			name:      "get_error",
			podName:   "p0",
			clientset: k8stest.NewErrorClientset("get", "pods", errors.New("mock error")),
			errorMsg:  fmt.Errorf("getting pod p0 in namespace test: mock error"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pod.NewClient(test.clientset, logging.New(io.Discard, 0))
			p, err := client.Get(t.Context(), test.podName, "test")
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if p == nil || p.Name != test.podName {
					t.Errorf("pod expected with name %q, got: %#v", test.podName, p)
				}
			} else {
				if err == nil {
					t.Fatalf("error not happened, expected: %s", test.errorMsg.Error())
				}
				if err.Error() != test.errorMsg.Error() {
					t.Errorf("error expected: %s, got: %s", test.errorMsg.Error(), err.Error())
				}
				if p != nil {
					t.Errorf("pod not expected")
				}
			}
		})
	}
}

func TestDeletePods(t *testing.T) {
	t.Parallel()
	beeLabels := map[string]string{"app": "bee"}

	terminating := newPod("p2", beeLabels)
	terminating.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}

	t.Run("deletes_non_terminating", func(t *testing.T) {
		cs := fake.NewClientset(
			newPod("p0", beeLabels),
			newPod("p1", beeLabels),
			terminating, // already terminating → skipped
		)
		client := pod.NewClient(cs, logging.New(io.Discard, 0))
		deleted, err := client.DeletePods(t.Context(), "test", "app=bee")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if deleted != 2 {
			t.Errorf("deleted count expected: 2, got: %d", deleted)
		}
	})

	t.Run("list_error", func(t *testing.T) {
		client := pod.NewClient(k8stest.NewErrorClientset("list", "pods", errors.New("mock error")), logging.New(io.Discard, 0))
		deleted, err := client.DeletePods(t.Context(), "test", "app=bee")
		if err == nil || err.Error() != "listing pods in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Errorf("deleted count expected: 0, got: %d", deleted)
		}
	})

	t.Run("delete_error", func(t *testing.T) {
		cs := k8stest.NewErrorClientset("delete", "pods", errors.New("mock error"), newPod("p0", beeLabels))
		client := pod.NewClient(cs, logging.New(io.Discard, 0))
		deleted, err := client.DeletePods(t.Context(), "test", "app=bee")
		if err == nil || err.Error() != "some pods failed to delete: [mock error]" {
			t.Errorf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Errorf("deleted count expected: 0, got: %d", deleted)
		}
	})
}

func TestGetControllingStatefulSet(t *testing.T) {
	t.Parallel()
	controller := true
	podWithOwner := func(kind string) *v1.Pod {
		p := newPod("p0", nil)
		p.OwnerReferences = []metav1.OwnerReference{
			{Kind: kind, Name: "sts-0", Controller: &controller},
		}
		return p
	}

	t.Run("controlled_by_statefulset", func(t *testing.T) {
		cs := fake.NewClientset(
			podWithOwner("StatefulSet"),
			&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "sts-0", Namespace: "test"}},
		)
		client := pod.NewClient(cs, logging.New(io.Discard, 0))
		sts, err := client.GetControllingStatefulSet(t.Context(), "p0", "test")
		if err != nil {
			t.Fatalf("error not expected, got: %s", err.Error())
		}
		if sts == nil || sts.Name != "sts-0" {
			t.Errorf("statefulset expected with name sts-0, got: %#v", sts)
		}
	})

	t.Run("no_owner_ref", func(t *testing.T) {
		client := pod.NewClient(fake.NewClientset(newPod("p0", nil)), logging.New(io.Discard, 0))
		_, err := client.GetControllingStatefulSet(t.Context(), "p0", "test")
		if err == nil || err.Error() != "pod p0 in namespace test is not controlled by a StatefulSet" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("wrong_owner_kind", func(t *testing.T) {
		client := pod.NewClient(fake.NewClientset(podWithOwner("ReplicaSet")), logging.New(io.Discard, 0))
		_, err := client.GetControllingStatefulSet(t.Context(), "p0", "test")
		if err == nil || err.Error() != "pod p0 in namespace test is not controlled by a StatefulSet" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pod_get_error", func(t *testing.T) {
		client := pod.NewClient(k8stest.NewErrorClientset("get", "pods", errors.New("mock error")), logging.New(io.Discard, 0))
		_, err := client.GetControllingStatefulSet(t.Context(), "p0", "test")
		if err == nil || err.Error() != "getting pod p0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("statefulset_get_error", func(t *testing.T) {
		// pod Get succeeds (seeded), but the StatefulSet Get reactor fails.
		cs := k8stest.NewErrorClientset("get", "statefulsets", errors.New("mock error"), podWithOwner("StatefulSet"))
		client := pod.NewClient(cs, logging.New(io.Discard, 0))
		_, err := client.GetControllingStatefulSet(t.Context(), "p0", "test")
		if err == nil || err.Error() != "getting StatefulSet sts-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestPodRecreationStateString(t *testing.T) {
	t.Parallel()
	testTable := []struct {
		state    pod.PodRecreationState
		expected string
	}{
		{pod.WaitingForDeletion, "WaitingForDeletion"},
		{pod.WaitingForCreation, "WaitingForCreation"},
		{pod.WaitingForRunning, "WaitingForRunning"},
		{pod.WaitingForCompletion, "WaitingForCompletion"},
		{pod.Completed, "Completed"},
		{pod.PodRecreationState(99), "Unknown"},
	}

	for _, test := range testTable {
		t.Run(test.expected, func(t *testing.T) {
			if got := test.state.String(); got != test.expected {
				t.Errorf("String() expected: %q, got: %q", test.expected, got)
			}
		})
	}
}

// podWatchClientset returns a fake clientset whose Pod Watch is served by w
// (and fails with watchErr if non-nil), per decisions §3.
func podWatchClientset(w *watch.RaceFreeFakeWatcher, watchErr error) kubernetes.Interface {
	cs := fake.NewClientset()
	cs.PrependWatchReactor("pods", k8stesting.DefaultWatchReactor(w, watchErr))
	return cs
}

// runningReadyPod is a Pod event payload that satisfies WatchNewRunning's
// running-and-ready filter.
func runningReadyPod(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"},
		Status: v1.PodStatus{
			PodIP: "10.0.0.1",
			Phase: v1.PodRunning,
			Conditions: []v1.PodCondition{
				{Type: v1.PodReady, Status: v1.ConditionTrue},
			},
		},
	}
}

func TestWatchNewRunning(t *testing.T) {
	t.Parallel()
	t.Run("watch_error", func(t *testing.T) {
		client := pod.NewClient(podWatchClientset(nil, errors.New("mock error")), logging.New(io.Discard, 0))
		err := client.WatchNewRunning(t.Context(), "test", "", make(chan *v1.Pod, 1))
		if err == nil || err.Error() != "getting pod events in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("context_cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		client := pod.NewClient(podWatchClientset(watch.NewRaceFreeFake(), nil), logging.New(io.Discard, 0))
		err := client.WatchNewRunning(ctx, "test", "", make(chan *v1.Pod, 1))
		if err == nil || err.Error() != "context canceled" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("channel_closed", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		w.Stop() // closes the result channel
		client := pod.NewClient(podWatchClientset(w, nil), logging.New(io.Discard, 0))
		err := client.WatchNewRunning(t.Context(), "test", "", make(chan *v1.Pod, 1))
		if err == nil || err.Error() != "watch channel closed" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("sends_ready_running_pod", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		// RaceFreeFake buffers events, so enqueue before the watch starts.
		w.Add(runningReadyPod("ignored")) // non-Modified type → ignored by the switch
		notRunning := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "test"}, Status: v1.PodStatus{Phase: v1.PodPending}}
		w.Modify(notRunning)            // Modified *Pod but no IP / not Running → filtered out
		w.Modify(runningReadyPod("p0")) // → sent to newPods

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		client := pod.NewClient(podWatchClientset(w, nil), logging.New(io.Discard, 0))
		newPods := make(chan *v1.Pod, 1)
		errCh := make(chan error, 1)
		go func() { errCh <- client.WatchNewRunning(ctx, "test", "", newPods) }()

		select {
		case got := <-newPods:
			if got == nil || got.Name != "p0" {
				t.Errorf("expected pod p0, got: %#v", got)
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for a running pod")
		}

		cancel() // stop the watch loop
		if err := <-errCh; err == nil || err.Error() != "context canceled" {
			t.Errorf("unexpected error after cancel: %v", err)
		}
	})
}

func TestWaitForRunning(t *testing.T) {
	t.Parallel()
	podInPhase := func(phase v1.PodPhase) *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test_pod", Namespace: "test"},
			Status:     v1.PodStatus{Phase: phase},
		}
	}

	t.Run("running", func(t *testing.T) {
		client := pod.NewClient(fake.NewClientset(podInPhase(v1.PodRunning)), logging.New(io.Discard, 0))
		if err := client.WaitForRunning(t.Context(), "test", "test_pod"); err != nil {
			t.Errorf("error not expected, got: %s", err.Error())
		}
	})

	t.Run("get_error", func(t *testing.T) {
		client := pod.NewClient(k8stest.NewErrorClientset("get", "pods", errors.New("mock error")), logging.New(io.Discard, 0))
		err := client.WaitForRunning(t.Context(), "test", "test_pod")
		if err == nil || err.Error() != "getting pod test_pod in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("bad_state_failed", func(t *testing.T) {
		client := pod.NewClient(fake.NewClientset(podInPhase(v1.PodFailed)), logging.New(io.Discard, 0))
		err := client.WaitForRunning(t.Context(), "test", "test_pod")
		if err == nil || err.Error() != "pod test_pod entered a bad state: Failed" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("bad_state_unknown", func(t *testing.T) {
		client := pod.NewClient(fake.NewClientset(podInPhase(v1.PodUnknown)), logging.New(io.Discard, 0))
		err := client.WaitForRunning(t.Context(), "test", "test_pod")
		if err == nil || err.Error() != "pod test_pod entered a bad state: Unknown" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("pending_polls_until_context_cancelled", func(t *testing.T) {
		// immediate=true runs the condition once (Pending → keep-polling branch)
		// before PollUntilContextCancel observes the cancelled context, so no 5s
		// poll interval elapses.
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		client := pod.NewClient(fake.NewClientset(podInPhase(v1.PodPending)), logging.New(io.Discard, 0))
		err := client.WaitForRunning(ctx, "test", "test_pod")
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})
}

func TestWaitForPodRecreationAndCompletion(t *testing.T) {
	t.Parallel()
	// lifecyclePod is the base pod (name "bee-0", one container "bee").
	lifecyclePod := func() *v1.Pod {
		return &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "bee-0", Namespace: "test"},
			Spec:       v1.PodSpec{Containers: []v1.Container{{Name: "bee"}}},
		}
	}
	runningPod := func() *v1.Pod {
		p := lifecyclePod()
		p.Status.ContainerStatuses = []v1.ContainerStatus{
			{Name: "bee", State: v1.ContainerState{Running: &v1.ContainerStateRunning{}}},
		}
		return p
	}
	terminatedPod := func(reason string, exitCode int32) *v1.Pod {
		p := lifecyclePod()
		p.Status.ContainerStatuses = []v1.ContainerStatus{
			{Name: "bee", State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{Reason: reason, ExitCode: exitCode}}},
		}
		return p
	}

	t.Run("completes_full_lifecycle", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		w.Delete(lifecyclePod())                // WaitingForDeletion -> WaitingForCreation
		w.Add(lifecyclePod())                   // -> WaitingForRunning
		w.Modify(runningPod())                  // -> WaitingForCompletion
		w.Modify(terminatedPod("Completed", 0)) // -> Completed (returns nil)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		client := pod.NewClient(podWatchClientset(w, nil), logging.New(io.Discard, 0))
		if err := client.WaitForPodRecreationAndCompletion(ctx, "test", "bee-0", "bee"); err != nil {
			t.Errorf("error not expected, got: %s", err.Error())
		}
	})

	t.Run("container_terminated_with_error", func(t *testing.T) {
		w := watch.NewRaceFreeFake()
		w.Delete(lifecyclePod())
		w.Add(lifecyclePod())
		w.Modify(runningPod())
		w.Modify(terminatedPod("Error", 1)) // -> returns an error from processEventInState

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		client := pod.NewClient(podWatchClientset(w, nil), logging.New(io.Discard, 0))
		err := client.WaitForPodRecreationAndCompletion(ctx, "test", "bee-0", "bee")
		if err == nil || err.Error() != "pod bee-0 container bee terminated with an error (ExitCode: 1)" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("skips_unrelated_events_and_uses_first_container", func(t *testing.T) {
		// containerName "" exercises the first-container fallback; the leading
		// events exercise the non-pod skip and the no-transition default.
		w := watch.NewRaceFreeFake()
		w.Modify(&v1.Service{})                 // not a *Pod -> skipped
		w.Modify(lifecyclePod())                // Modified while WaitingForDeletion -> no transition
		w.Delete(lifecyclePod())                // -> WaitingForCreation
		w.Add(lifecyclePod())                   // -> WaitingForRunning
		w.Modify(runningPod())                  // -> WaitingForCompletion
		w.Modify(terminatedPod("Completed", 0)) // -> Completed

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		client := pod.NewClient(podWatchClientset(w, nil), logging.New(io.Discard, 0))
		if err := client.WaitForPodRecreationAndCompletion(ctx, "test", "bee-0", ""); err != nil {
			t.Errorf("error not expected, got: %s", err.Error())
		}
	})

	t.Run("watch_error", func(t *testing.T) {
		client := pod.NewClient(podWatchClientset(nil, errors.New("mock error")), logging.New(io.Discard, 0))
		err := client.WaitForPodRecreationAndCompletion(t.Context(), "test", "bee-0", "bee")
		if err == nil || err.Error() != "getting watch for pod bee-0 in namespace test: mock error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		// No events are sent; the watchCtx deadline fires while still in the
		// initial state.
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		client := pod.NewClient(podWatchClientset(watch.NewRaceFreeFake(), nil), logging.New(io.Discard, 0))
		err := client.WaitForPodRecreationAndCompletion(ctx, "test", "bee-0", "bee")
		if err == nil || err.Error() != "timed out waiting for pod bee-0 to complete (current state: WaitingForDeletion)" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
