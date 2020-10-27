package persistentvolumeclaims

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeClaims represents Kubernetes PersistentVolumeClaims
type PersistentVolumeClaims []PersistentVolumeClaim

// ToK8S converts PersistentVolumeClaims to Kuberntes client objects
func (ps PersistentVolumeClaims) ToK8S() (l []v1.PersistentVolumeClaim) {
	l = make([]v1.PersistentVolumeClaim, 0, len(ps))

	for _, p := range ps {
		l = append(l, p.ToK8S())
	}

	return
}

// PersistentVolumeClaim represents Kubernetes PersistentVolumeClaim
type PersistentVolumeClaim struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Spec        PersistentVolumeClaimSpec
}

// ToK8S converts PersistentVolumeClaim to Kuberntes client object
func (pvc PersistentVolumeClaim) ToK8S() v1.PersistentVolumeClaim {
	return v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvc.Name,
			Namespace:   pvc.Namespace,
			Annotations: pvc.Annotations,
			Labels:      pvc.Labels,
		},
		Spec: pvc.Spec.toK8S(),
	}
}

// PersistentVolumeClaimSpec represents Kubernetes PersistentVolumeClaimSpec
type PersistentVolumeClaimSpec struct {
	Name           string
	AccessModes    AccessModes
	DataSource     DataSource
	RequestStorage string
	Selector       Selector
	StorageClass   string
	VolumeMode     string
	VolumeName     string
}

// toK8S converts PersistentVolumeClaimSpec to Kuberntes client object
func (pvcs PersistentVolumeClaimSpec) toK8S() v1.PersistentVolumeClaimSpec {
	return v1.PersistentVolumeClaimSpec{
		AccessModes: pvcs.AccessModes.toK8S(),
		DataSource:  pvcs.DataSource.toK8S(),
		Resources: v1.ResourceRequirements{
			Requests: func() v1.ResourceList {
				m := map[v1.ResourceName]resource.Quantity{}
				if len(pvcs.RequestStorage) > 0 {
					m[v1.ResourceStorage] = resource.MustParse(pvcs.RequestStorage)
				}
				return m
			}(),
		},
		Selector:         pvcs.Selector.toK8S(),
		StorageClassName: &pvcs.StorageClass,
		VolumeName:       pvcs.VolumeName,
		VolumeMode: func() *v1.PersistentVolumeMode {
			if pvcs.VolumeMode == "Block" || pvcs.VolumeMode == "block" {
				m := v1.PersistentVolumeMode(v1.PersistentVolumeBlock)
				return &m
			}
			m := v1.PersistentVolumeMode(v1.PersistentVolumeFilesystem)
			return &m
		}(),
	}
}
