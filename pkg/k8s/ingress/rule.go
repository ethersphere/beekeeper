package ingress

import v1 "k8s.io/api/networking/v1"

// Rules represents Kubernetes IngressRules
type Rules []Rule

// toK8S converts Rules to Kubernetes client objects
func (rs Rules) toK8S() (l []v1.IngressRule) {
	if len(rs) > 0 {
		l = make([]v1.IngressRule, 0, len(rs))

		for _, r := range rs {
			l = append(l, r.toK8S())
		}
	}
	return
}

// Rule represents Kubernetes IngressRule
type Rule struct {
	Host  string
	Paths Paths
}

// toK8S converts Rule to Kubernetes client object
func (r *Rule) toK8S() (rule v1.IngressRule) {
	return v1.IngressRule{
		Host: r.Host,
		IngressRuleValue: v1.IngressRuleValue{
			HTTP: &v1.HTTPIngressRuleValue{
				Paths: r.Paths.toK8S(),
			},
		},
	}
}

// Paths represents service's HTTPIngressPaths
type Paths []Path

// toK8S converts Paths to Kubernetes client objects
func (ps Paths) toK8S() (l []v1.HTTPIngressPath) {
	if len(ps) > 0 {
		l = make([]v1.HTTPIngressPath, 0, len(ps))

		for _, p := range ps {
			l = append(l, p.toK8S())
		}
	}
	return
}

// Path represents service's HTTPIngressPath
type Path struct {
	Backend  Backend
	Path     string
	PathType string
}

// toK8S converts Path to Kubernetes client object
func (p *Path) toK8S() (h v1.HTTPIngressPath) {
	pt := v1.PathType(p.PathType)
	return v1.HTTPIngressPath{
		Backend:  p.Backend.toK8S(),
		Path:     p.Path,
		PathType: &pt,
	}
}
