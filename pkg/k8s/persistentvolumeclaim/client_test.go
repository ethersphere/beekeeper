package persistentvolumeclaim

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
		pvcName   string
		options   Options
		clientset kubernetes.Interface
		errorMsg  error
	}{
		{
			name:      "create_pvc",
			pvcName:   "test_pvc",
			clientset: fake.NewSimpleClientset(),
			options: Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_1"},
				Labels:      map[string]string{"label_1": "label_value_1"},
				Spec: PersistentVolumeClaimSpec{
					AccessModes:    []AccessMode{"1", "2"},
					RequestStorage: "1Gi",
					Selector: Selector{
						MatchLabels: map[string]string{"label_1": "label_value_1"},
						MatchExpressions: []LabelSelectorRequirement{
							{
								Key:      "label_1",
								Operator: "==",
								Values:   []string{"label_value_1"},
							},
						},
					},
					VolumeName: "volume_1",
				},
			},
		},
		{
			name:      "create_pvc_spec_volume_mode_Block",
			pvcName:   "test_pvc",
			clientset: fake.NewSimpleClientset(),
			options: Options{
				Spec: PersistentVolumeClaimSpec{
					VolumeMode: "Block",
				},
			},
		},
		{
			name:      "create_pvc_spec_volume_mode_block",
			pvcName:   "test_pvc",
			clientset: fake.NewSimpleClientset(),
			options: Options{
				Spec: PersistentVolumeClaimSpec{
					VolumeMode: "block",
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
			options: Options{
				Annotations: map[string]string{"annotation_1": "annotation_value_X", "annotation_2": "annotation_value_2"},
			},
		},
		{
			name:      "create_error",
			pvcName:   "create_bad",
			clientset: mocks.NewClientset(),
			errorMsg:  fmt.Errorf("creating pvc create_bad in namespace test: mock error: cannot create pvc"),
		},
		{
			name:      "update_error",
			pvcName:   "update_bad",
			clientset: mocks.NewClientset(),
			errorMsg:  fmt.Errorf("updating pvc update_bad in namespace test: mock error: cannot update pvc"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
			response, err := client.Set(context.Background(), test.pvcName, "test", test.options)
			if test.errorMsg == nil {
				if err != nil {
					t.Errorf("error not expected, got: %s", err.Error())
				}
				if response == nil {
					t.Fatalf("response is expected")
				}

				expectedSpec := test.options.Spec.toK8S()

				if test.options.Spec.VolumeMode == "Block" || test.options.Spec.VolumeMode == "block" {
					m := v1.PersistentVolumeBlock
					expectedSpec.VolumeMode = &m
				} else {
					m := v1.PersistentVolumeFilesystem
					expectedSpec.VolumeMode = &m
				}

				expected := &v1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:        test.pvcName,
						Namespace:   "test",
						Annotations: test.options.Annotations,
						Labels:      test.options.Labels,
					},
					Spec: expectedSpec,
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
			pvcName:   "delete_bad",
			clientset: mocks.NewClientset(),
			errorMsg:  fmt.Errorf("deleting pvc delete_bad in namespace test: mock error: cannot delete pvc"),
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.clientset)
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
