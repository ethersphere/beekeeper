package pod

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TopologySpreadConstraints represents Kubernetes TopologySpreadConstraints
type TopologySpreadConstraints []TopologySpreadConstraint

// toK8S converts TopologySpreadConstraints to Kubernetes client objects
func (tscs TopologySpreadConstraints) toK8S() (l []v1.TopologySpreadConstraint) {
	if len(tscs) > 0 {
		l = make([]v1.TopologySpreadConstraint, 0, len(tscs))
		for _, t := range tscs {
			l = append(l, t.toK8S())
		}
	}
	return l
}

// TopologySpreadConstraint represents Kubernetes TopologySpreadConstraint
type TopologySpreadConstraint struct {
	MaxSkew           int32
	TopologyKey       string
	WhenUnsatisfiable string
	LabelSelector     map[string]string
}

// toK8S converts TopologySpreadConstraint to Kubernetes client object
func (tsc *TopologySpreadConstraint) toK8S() v1.TopologySpreadConstraint {
	return v1.TopologySpreadConstraint{
		MaxSkew:           tsc.MaxSkew,
		TopologyKey:       tsc.TopologyKey,
		WhenUnsatisfiable: v1.UnsatisfiableConstraintAction(tsc.WhenUnsatisfiable),
		LabelSelector:     &metav1.LabelSelector{MatchLabels: tsc.LabelSelector},
	}
}
