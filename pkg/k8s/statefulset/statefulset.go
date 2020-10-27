package statefulset

import (
	pvc "github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaims"
	"github.com/ethersphere/beekeeper/pkg/k8s/pods"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSetSpec represents Kubernetes StatefulSetSpec
type StatefulSetSpec struct {
	PodManagementPolicy  string
	Replicas             int32
	RevisionHistoryLimit int32
	Selector             map[string]string
	ServiceName          string
	Template             pods.PodTemplateSpec
	UpdateStrategy       UpdateStrategy
	VolumeClaimTemplates pvc.PersistentVolumeClaims
}

// ToK8S converts StatefulSetSpec to Kuberntes client object
func (s StatefulSetSpec) ToK8S() appsv1.StatefulSetSpec {
	return appsv1.StatefulSetSpec{
		PodManagementPolicy:  appsv1.PodManagementPolicyType(s.PodManagementPolicy),
		Replicas:             &s.Replicas,
		RevisionHistoryLimit: &s.RevisionHistoryLimit,
		Selector:             &metav1.LabelSelector{MatchLabels: s.Selector},
		ServiceName:          s.ServiceName,
		Template:             s.Template.ToK8S(),
		UpdateStrategy:       s.UpdateStrategy.toK8S(),
		VolumeClaimTemplates: s.VolumeClaimTemplates.ToK8S(),
	}
}

// UpdateStrategy represents Kubernetes StatefulSetUpdateStrategy
type UpdateStrategy struct {
	Type                   string
	RollingUpdatePartition int32
}

// toK8S converts UpdateStrategy to Kuberntes client object
func (u UpdateStrategy) toK8S() appsv1.StatefulSetUpdateStrategy {
	if u.Type == "OnDelete" {
		return appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.OnDeleteStatefulSetStrategyType,
		}
	}

	return appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
			Partition: &u.RollingUpdatePartition,
		},
	}
}
