package persistentvolumeclaim_test

import (
	"reflect"
	"testing"

	pvc "github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaim"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToK8s(t *testing.T) {
	testTable := []struct {
		name         string
		pvcs         pvc.PersistentVolumeClaims
		expectedPvcs []v1.PersistentVolumeClaim
	}{
		{
			name: "all_and_volume_mode_no_block",
			pvcs: pvc.PersistentVolumeClaims{
				{
					Name:        "pvc",
					Namespace:   "test",
					Annotations: map[string]string{"annotation_1": "annotation_value_1"},
					Labels:      map[string]string{"label_1": "label_value_1"},
					Spec: pvc.PersistentVolumeClaimSpec{
						AccessModes:    []pvc.AccessMode{"1", "2"},
						RequestStorage: "1Gi",
						Selector: pvc.Selector{
							MatchLabels: map[string]string{"label_1": "label_value_1"},
							MatchExpressions: []pvc.LabelSelectorRequirement{
								{
									Key:      "label_1",
									Operator: "==",
									Values:   []string{"label_value_1"},
								},
							},
						},
						VolumeName: "volume_1",
						Name:       "spec",
						DataSource: pvc.DataSource{
							APIGroup: "APIGroup",
							Kind:     "Kind",
							Name:     "Name",
						},
						StorageClass: "StorageClass",
						VolumeMode:   "not_block",
					},
				},
			},
			expectedPvcs: []v1.PersistentVolumeClaim{
				{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "pvc",
						Namespace:   "test",
						Annotations: map[string]string{"annotation_1": "annotation_value_1"},
						Labels:      map[string]string{"label_1": "label_value_1"},
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{"1", "2"},
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"label_1": "label_value_1"},
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "label_1",
									Operator: "==",
									Values:   []string{"label_value_1"},
								},
							},
						},
						Resources: v1.ResourceRequirements{
							Limits:   nil,
							Requests: map[v1.ResourceName]resource.Quantity{v1.ResourceStorage: resource.MustParse("1Gi")},
						},
						VolumeName:       "volume_1",
						StorageClassName: getAddress("StorageClass"),
						DataSource: &v1.TypedLocalObjectReference{
							APIGroup: getAddress("APIGroup"),
							Kind:     "Kind",
							Name:     "Name",
						},
						VolumeMode: func() *v1.PersistentVolumeMode {
							m := v1.PersistentVolumeFilesystem
							return &m
						}(),
					},
					Status: v1.PersistentVolumeClaimStatus{},
				},
			},
		},
		{
			name: "default_and_volume_mode_block",
			pvcs: pvc.PersistentVolumeClaims{
				{
					Spec: pvc.PersistentVolumeClaimSpec{
						VolumeMode: "block",
					},
				},
			},
			expectedPvcs: []v1.PersistentVolumeClaim{
				{
					Spec: v1.PersistentVolumeClaimSpec{
						Selector: &metav1.LabelSelector{},
						Resources: v1.ResourceRequirements{
							Limits:   nil,
							Requests: map[v1.ResourceName]resource.Quantity{},
						},
						VolumeMode: func() *v1.PersistentVolumeMode {
							m := v1.PersistentVolumeBlock
							return &m
						}(),
						StorageClassName: getAddress(""),
						DataSource: &v1.TypedLocalObjectReference{
							APIGroup: getAddress(""),
						},
					},
				},
			},
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			pvcs := test.pvcs.ToK8S()
			if !reflect.DeepEqual(pvcs, test.expectedPvcs) {
				t.Errorf("response expected: %#v, got: %#v", test.expectedPvcs, pvcs)
			}
		})
	}
}

func getAddress(value string) *string {
	return &value
}
