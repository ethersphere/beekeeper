package statefulset

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeClaim ...
type PersistentVolumeClaim struct {
	Name           string
	AccessModes    []AccessMode
	DataSource     DataSource
	RequestStorage Request
	Selector       Selector
	StorageClass   string
	VolumeMode     string
	VolumeName     string
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
			DataSource:  p.DataSource.toK8S(),
			Resources: v1.ResourceRequirements{
				Requests: p.RequestStorage.toK8S(),
			},
			Selector:         p.Selector.toK8S(),
			StorageClassName: &p.StorageClass,
			VolumeName:       p.VolumeName,
			VolumeMode: func() *v1.PersistentVolumeMode {
				pvm := v1.PersistentVolumeMode(p.VolumeMode)
				return &pvm
			}(),
		},
	}
}

func persistentVolumeClaimsToK8S(persistentVolumeClaims []PersistentVolumeClaim, namespace string, annotations, labels map[string]string) (pvcs []v1.PersistentVolumeClaim) {
	for _, pvc := range persistentVolumeClaims {
		pvcs = append(pvcs, pvc.toK8S(namespace, annotations, labels))
	}
	return
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

// DataSource ...
type DataSource struct {
	APIGroup string
	Kind     string
	Name     string
}

func (d DataSource) toK8S() *v1.TypedLocalObjectReference {
	return &v1.TypedLocalObjectReference{
		APIGroup: &d.APIGroup,
		Kind:     d.Kind,
		Name:     d.Name,
	}
}

// Selector ...
type Selector struct {
	MatchLabels      map[string]string
	MatchExpressions []LabelSelectorRequirement
}

func (s Selector) toK8S() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels:      s.MatchLabels,
		MatchExpressions: labelSelectorRequirementsToK8S(s.MatchExpressions),
	}
}

// LabelSelectorRequirement ...
type LabelSelectorRequirement struct {
	Key      string
	Operator string
	Values   []string
}

func (l LabelSelectorRequirement) toK8S() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      l.Key,
		Operator: metav1.LabelSelectorOperator(l.Operator),
		Values:   l.Values,
	}
}

func labelSelectorRequirementsToK8S(labelSelectorRequirements []LabelSelectorRequirement) (l []metav1.LabelSelectorRequirement) {
	for _, lsr := range labelSelectorRequirements {
		l = append(l, lsr.toK8S())
	}
	return
}
