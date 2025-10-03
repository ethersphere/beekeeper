package pod

import v1 "k8s.io/api/core/v1"

// PodReadinessGates represents Kubernetes PodReadinessGates
type PodReadinessGates []PodReadinessGate

// toK8S converts PodReadinessGates to Kubernetes client objects
func (prgs PodReadinessGates) toK8S() (l []v1.PodReadinessGate) {
	if len(prgs) > 0 {
		l = make([]v1.PodReadinessGate, 0, len(prgs))
		for _, g := range prgs {
			l = append(l, g.toK8S())
		}
	}
	return l
}

// PodReadinessGate represents Kubernetes PodReadinessGate
type PodReadinessGate struct {
	ConditionType string
}

// toK8S converts PodReadinessGate to Kubernetes client object
func (prg *PodReadinessGate) toK8S() v1.PodReadinessGate {
	return v1.PodReadinessGate{
		ConditionType: v1.PodConditionType(prg.ConditionType),
	}
}
