package pods

import v1 "k8s.io/api/core/v1"

// Tolerations represents Kubernetes Tolerations
type Tolerations []Toleration

// toK8S converts Tolerations to Kuberntes client object
func (ts Tolerations) toK8S() (l []v1.Toleration) {
	l = make([]v1.Toleration, 0, len(ts))

	for _, p := range ts {
		l = append(l, p.toK8S())
	}

	return
}

// Toleration represents Kubernetes Toleration
type Toleration struct {
	Key               string
	Operator          string
	Value             string
	Effect            string
	TolerationSeconds int64
}

// toK8S converts Toleration to Kuberntes client object
func (t Toleration) toK8S() v1.Toleration {
	return v1.Toleration{
		Key:               t.Key,
		Operator:          v1.TolerationOperator(t.Operator),
		Value:             t.Value,
		Effect:            v1.TaintEffect(t.Effect),
		TolerationSeconds: &t.TolerationSeconds,
	}
}
