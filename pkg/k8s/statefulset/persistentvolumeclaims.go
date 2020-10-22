package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeClaim ...
type PersistentVolumeClaim struct {
	Name           string
	AccessModes    []AccessMode
	RequestStorage string
	StorageClass   string
}

func (p PersistentVolumeClaim) toK8S(namespace string, annotations, labels map[string]string) v1.PersistentVolumeClaim {
	return v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        p.Name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: accessModesToK8S(p.AccessModes),
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(p.RequestStorage),
				},
			},
			StorageClassName: &p.StorageClass,
		},
	}
}

// AccessMode ...
type AccessMode string

func (a AccessMode) toK8S() v1.PersistentVolumeAccessMode {
	return v1.PersistentVolumeAccessMode(a)
}

func accessModesToK8S(accessModes []AccessMode) (ams []v1.PersistentVolumeAccessMode) {
	for _, am := range accessModes {
		ams = append(ams, am.toK8S())
	}
	return
}
