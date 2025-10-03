package persistentvolumeclaim

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Selector represents Kubernetes LabelSelector
type Selector struct {
	MatchLabels      map[string]string
	MatchExpressions LabelSelectorRequirements
}

// toK8S converts Selector to Kubernetes client object
func (s *Selector) toK8S() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels:      s.MatchLabels,
		MatchExpressions: s.MatchExpressions.toK8S(),
	}
}

// LabelSelectorRequirements represents Kubernetes LabelSelectorRequirements
type LabelSelectorRequirements []LabelSelectorRequirement

// toK8S converts LabelSelectorRequirements to Kubernetes client object
func (lsrs LabelSelectorRequirements) toK8S() (l []metav1.LabelSelectorRequirement) {
	if len(lsrs) > 0 {
		l = make([]metav1.LabelSelectorRequirement, 0, len(lsrs))
		for _, lsr := range lsrs {
			l = append(l, lsr.toK8S())
		}
	}
	return l
}

// LabelSelectorRequirement represents Kubernetes LabelSelectorRequirement
type LabelSelectorRequirement struct {
	Key      string
	Operator string
	Values   []string
}

// toK8S converts LabelSelectorRequirement to Kubernetes client object
func (l *LabelSelectorRequirement) toK8S() metav1.LabelSelectorRequirement {
	return metav1.LabelSelectorRequirement{
		Key:      l.Key,
		Operator: metav1.LabelSelectorOperator(l.Operator),
		Values:   l.Values,
	}
}
