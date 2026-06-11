package persistentvolumeclaim_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	pvc "github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaim"
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
		name         string
		pvcName      string
		options      pvc.Options
		clientset    kubernetes.Interface
		expectedSpec v1.PersistentVolumeClaimSpec
		errorMsg     error
	}{
		{
			name:      "create_pvc_default_spec",
			pvcName:   "test_pvc",
			clientset: fake.NewSimpleClientset(),
			options: pvc.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
			},
			expectedSpec: v1.PersistentVolumeClaimSpec{
				Selector: &metav1.LabelSelector{},
				VolumeMode: func() *v1.PersistentVolumeMode {
					m := v1.PersistentVolumeFilesystem
					return &m
				}(),
				StorageClassName: new(""),
				DataSource: &v1.TypedLocalObjectReference{
					APIGroup: new(""),
				},
			},
		},
		{
			name:    "update_pvc",
			pvcName: "test_pvc",
			clientset: fake.NewSimpleClientset(&v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test_pvc",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
				},
			}),
			options: pvc.Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
			expectedSpec: v1.PersistentVolumeClaimSpec{
				Selector: &metav1.LabelSelector{},
				VolumeMode: func() *v1.PersistentVolumeMode {
					m := v1.PersistentVolumeFilesystem
					return &m
				}(),
				StorageClassName: new(""),
				DataSource: &v1.TypedLocalObjectReference{
					APIGroup: new(""),
				},
			},
		},
		{
			name:    "create_error",
			pvcName: "test_pvc",
			// No object seeded, so Update returns NotFound and Set falls through
			// to Create, which the reactor fails.
			clientset: newErrorClientset("create", "persistentvolumeclaims", errors.New("mock error: cannot create pvc")),
			errorMsg:  fmt.Errorf("creating pvc test_pvc in namespace test: mock error: cannot create pvc"),
		},
		{
			name:      "update_error",
			pvcName:   "test_pvc",
			clientset: newErrorClientset("update", "persistentvolumeclaims", errors.New("mock error: cannot update pvc")),
			errorMsg:  fmt.Errorf("updating pvc test_pvc in namespace test: mock error: cannot update pvc"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pvc.NewClient(test.clientset)
			response, err := client.Set(t.Context(), test.pvcName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expected := &v1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.pvcName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: test.expectedSpec,
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
		pvcName   string
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:    "delete_pvc",
			pvcName: "test_pvc",
			clientset: fake.NewSimpleClientset(&v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_pvc",
					Namespace: "test",
				},
			}),
		},
		{
			name:    "delete_not_found",
			pvcName: "test_pvc_not_found",
			clientset: fake.NewSimpleClientset(&v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_pvc",
					Namespace: "test",
				},
			}),
		},
		{
			name:      "delete_error",
			pvcName:   "test_pvc",
			clientset: newErrorClientset("delete", "persistentvolumeclaims", errors.New("mock error: cannot delete pvc")),
			errorMsg:  fmt.Errorf("deleting pvc test_pvc in namespace test: mock error: cannot delete pvc"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pvc.NewClient(test.clientset)
			err := client.Delete(t.Context(), test.pvcName, "test")
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
