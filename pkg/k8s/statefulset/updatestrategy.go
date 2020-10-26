package statefulset

import appsv1 "k8s.io/api/apps/v1"

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
