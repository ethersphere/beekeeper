package pod

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TopologySpreadConstraints represents Kubernetes TopologySpreadConstraints
type TopologySpreadConstraints []TopologySpreadConstraint

// toK8S converts TopologySpreadConstraints to Kuberntes client objects
func (tscs TopologySpreadConstraints) toK8S() (l []v1.TopologySpreadConstraint) {
	l = make([]v1.TopologySpreadConstraint, 0, len(tscs))

	for _, t := range tscs {
		l = append(l, t.toK8S())
	}

	return
}

// TopologySpreadConstraint represents Kubernetes TopologySpreadConstraint
type TopologySpreadConstraint struct {
	MaxSkew           int32
	TopologyKey       string
	WhenUnsatisfiable string
	LabelSelector     map[string]string
}

// toK8S converts TopologySpreadConstraint to Kuberntes client object
func (tsc TopologySpreadConstraint) toK8S() v1.TopologySpreadConstraint {
	return v1.TopologySpreadConstraint{
		MaxSkew:           tsc.MaxSkew,
		TopologyKey:       tsc.TopologyKey,
		WhenUnsatisfiable: v1.UnsatisfiableConstraintAction(tsc.WhenUnsatisfiable),
		LabelSelector:     &metav1.LabelSelector{MatchLabels: tsc.LabelSelector},
	}
}
