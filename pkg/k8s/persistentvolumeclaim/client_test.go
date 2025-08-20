package persistentvolumeclaim_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	mock "github.com/ethersphere/beekeeper/pkg/k8s/mocks"
	pvc "github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaim"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSet(t *testing.T) {
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
				StorageClassName: getAddress(""),
				DataSource: &v1.TypedLocalObjectReference{
					APIGroup: getAddress(""),
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
				StorageClassName: getAddress(""),
				DataSource: &v1.TypedLocalObjectReference{
					APIGroup: getAddress(""),
				},
			},
		},
		{
			name:      "create_error",
			pvcName:   mock.CreateBad,
			clientset: mock.NewClientset(),
			errorMsg:  fmt.Errorf("creating pvc create_bad in namespace test: mock error: cannot create pvc"),
		},
		{
			name:      "update_error",
			pvcName:   mock.UpdateBad,
			clientset: mock.NewClientset(),
			errorMsg:  fmt.Errorf("updating pvc update_bad in namespace test: mock error: cannot update pvc"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pvc.NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.pvcName, "test", test.options)
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
			pvcName:   mock.DeleteBad,
			clientset: mock.NewClientset(),
			errorMsg:  fmt.Errorf("deleting pvc delete_bad in namespace test: mock error: cannot delete pvc"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := pvc.NewClient(test.clientset)
			err := client.Delete(context.Background(), test.pvcName, "test")
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
