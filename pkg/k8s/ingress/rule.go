package ingress

import (
	ev1b1 "k8s.io/api/extensions/v1beta1"
)

// Rules represents Kubernetes IngressRules
type Rules []Rule

// toK8S converts Rules to Kuberntes client objects
func (rs Rules) toK8S() (l []ev1b1.IngressRule) {
	l = make([]ev1b1.IngressRule, 0, len(rs))

	for _, r := range rs {
		l = append(l, r.toK8S())
	}

	return
}

// Rule represents Kubernetes IngressRule
type Rule struct {
	Host  string
	Paths Paths
}

// toK8S converts Rule to Kuberntes client object
func (r Rule) toK8S() (rule ev1b1.IngressRule) {
	return ev1b1.IngressRule{
		Host: r.Host,
		IngressRuleValue: ev1b1.IngressRuleValue{
			HTTP: &ev1b1.HTTPIngressRuleValue{
				Paths: r.Paths.toK8S(),
			},
		},
	}
}

// Paths represents service's HTTPIngressPaths
type Paths []Path

// toK8S converts Paths to Kuberntes client objects
func (ps Paths) toK8S() (l []ev1b1.HTTPIngressPath) {
	l = make([]ev1b1.HTTPIngressPath, 0, len(ps))

	for _, p := range ps {
		l = append(l, p.toK8S())
	}

	return
}

// Path represents service's HTTPIngressPath
type Path struct {
	Backend Backend
	Path    string
}

// toK8S converts Path to Kuberntes client object
func (p Path) toK8S() (h ev1b1.HTTPIngressPath) {
	return ev1b1.HTTPIngressPath{
		Backend: p.Backend.toK8S(),
		Path:    p.Path,
	}
}
