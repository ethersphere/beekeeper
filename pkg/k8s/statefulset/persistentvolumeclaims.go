package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeClaims represents Kubernetes PersistentVolumeClaims
type PersistentVolumeClaims []PersistentVolumeClaim

// toK8S converts PersistentVolumeClaims to Kuberntes client objects
func (pvcs PersistentVolumeClaims) toK8S(namespace string, annotations, labels map[string]string) (l []v1.PersistentVolumeClaim) {
	l = make([]v1.PersistentVolumeClaim, 0, len(pvcs))

	for _, p := range pvcs {
		l = append(l, p.toK8S(namespace, annotations, labels))
	}

	return
}

// PersistentVolumeClaim represents Kubernetes PersistentVolumeClaim
type PersistentVolumeClaim struct {
	Name           string
	AccessModes    AccessModes
	DataSource     DataSource
	RequestStorage string
	Selector       Selector
	StorageClass   string
	VolumeMode     string
	VolumeName     string
}

// toK8S converts PersistentVolumeClaim to Kuberntes client objects
func (pvc PersistentVolumeClaim) toK8S(namespace string, annotations, labels map[string]string) v1.PersistentVolumeClaim {
	return v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvc.Name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: pvc.AccessModes.toK8S(),
			DataSource:  pvc.DataSource.toK8S(),
			Resources: v1.ResourceRequirements{
				Requests: func() v1.ResourceList {
					m := map[v1.ResourceName]resource.Quantity{}
					if len(pvc.RequestStorage) > 0 {
						m[v1.ResourceStorage] = resource.MustParse(pvc.RequestStorage)
					}
					return m
				}(),
			},
			Selector:         pvc.Selector.toK8S(),
			StorageClassName: &pvc.StorageClass,
			VolumeName:       pvc.VolumeName,
			VolumeMode: func() *v1.PersistentVolumeMode {
				if pvc.VolumeMode == "Block" || pvc.VolumeMode == "block" {
					m := v1.PersistentVolumeMode(v1.PersistentVolumeBlock)
					return &m
				}
				m := v1.PersistentVolumeMode(v1.PersistentVolumeFilesystem)
				return &m
			}(),
		},
	}
}

// AccessModes represents Kubernetes AccessModes
type AccessModes []AccessMode

// toK8S converts AccessModes to Kuberntes client objects
func (ams AccessModes) toK8S() (l []v1.PersistentVolumeAccessMode) {
	l = make([]v1.PersistentVolumeAccessMode, 0, len(ams))

	for _, am := range ams {
		l = append(l, am.toK8S())
	}

	return
}

// AccessMode represents Kubernetes AccessMode
type AccessMode string

// toK8S converts AccessMode to Kuberntes client object
func (a AccessMode) toK8S() v1.PersistentVolumeAccessMode {
	return v1.PersistentVolumeAccessMode(a)
}

// DataSource represents Kubernetes DataSource
type DataSource struct {
	APIGroup string
	Kind     string
	Name     string
}

// toK8S converts DataSource to Kuberntes client object
func (d DataSource) toK8S() *v1.TypedLocalObjectReference {
	return &v1.TypedLocalObjectReference{
		APIGroup: &d.APIGroup,
		Kind:     d.Kind,
		Name:     d.Name,
	}
}

// Selector represents Kubernetes LabelSelector
type Selector struct {
	MatchLabels      map[string]string
	MatchExpressions LabelSelectorRequirements
}

// toK8S converts Selector to Kuberntes client object
func (s Selector) toK8S() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels:      s.MatchLabels,
		MatchExpressions: s.MatchExpressions.toK8S(),
	}
}

// LabelSelectorRequirements represents Kubernetes LabelSelectorRequirements
type LabelSelectorRequirements []LabelSelectorRequirement

// toK8S converts LabelSelectorRequirements to Kuberntes client object
func (lsrs LabelSelectorRequirements) toK8S() (l []metav1.LabelSelectorRequirement) {
	l = make([]metav1.LabelSelectorRequirement, 0, len(lsrs))

	for _, lsr := range lsrs {
		l = append(l, lsr.toK8S())
	}

	return
}

// LabelSelectorRequirement represents Kubernetes LabelSelectorRequirement
type LabelSelectorRequirement struct {
	Key      string
	Operator string
	Values   []string
}

// toK8S converts LabelSelectorRequirement to Kuberntes client object
func (l LabelSelectorRequirement) toK8S() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      l.Key,
		Operator: metav1.LabelSelectorOperator(l.Operator),
		Values:   l.Values,
	}
}
