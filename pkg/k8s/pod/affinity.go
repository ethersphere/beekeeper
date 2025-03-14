package pod

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Affinity represents Kubernetes Affinity
type Affinity struct {
	NodeAffinity    *NodeAffinity
	PodAffinity     *PodAffinity
	PodAntiAffinity *PodAntiAffinity
}

func (a *Affinity) toK8S() *v1.Affinity {
	r := v1.Affinity{}
	if a.NodeAffinity != nil {
		r.NodeAffinity = a.NodeAffinity.toK8S()
	}
	if a.PodAffinity != nil {
		r.PodAffinity = a.PodAffinity.toK8S()
	}
	if a.PodAntiAffinity != nil {
		r.PodAntiAffinity = a.PodAntiAffinity.toK8S()
	}
	return &r
}

// NodeAffinity represents Kubernetes NodeAffinity
type NodeAffinity struct {
	PreferredDuringSchedulingIgnoredDuringExecution PreferredSchedulingTerms
	RequiredDuringSchedulingIgnoredDuringExecution  NodeSelector
}

// toK8S converts NodeAffinity to Kubernetes client object
func (na *NodeAffinity) toK8S() *v1.NodeAffinity {
	return &v1.NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: na.PreferredDuringSchedulingIgnoredDuringExecution.toK8S(),
		RequiredDuringSchedulingIgnoredDuringExecution:  na.RequiredDuringSchedulingIgnoredDuringExecution.toK8S(),
	}
}

// PreferredSchedulingTerms represents Kubernetes PreferredSchedulingTerms
type PreferredSchedulingTerms []PreferredSchedulingTerm

func (psts PreferredSchedulingTerms) toK8S() (l []v1.PreferredSchedulingTerm) {
	if len(psts) > 0 {
		l = make([]v1.PreferredSchedulingTerm, 0, len(psts))
		for _, p := range psts {
			l = append(l, p.toK8S())
		}
	}
	return
}

// PreferredSchedulingTerm represents Kubernetes PreferredSchedulingTerm
type PreferredSchedulingTerm struct {
	Preference NodeSelectorTerm
	Weight     int32
}

// toK8S converts PreferredSchedulingTerm to Kubernetes client object
func (pst *PreferredSchedulingTerm) toK8S() v1.PreferredSchedulingTerm {
	return v1.PreferredSchedulingTerm{
		Preference: pst.Preference.toK8S(),
		Weight:     pst.Weight,
	}
}

// NodeSelector represents Kubernetes NodeSelector
type NodeSelector struct {
	NodeSelectorTerms NodeSelectorTerms
}

// toK8S converts NodeSelector to Kubernetes client object
func (ns *NodeSelector) toK8S() *v1.NodeSelector {
	return &v1.NodeSelector{
		NodeSelectorTerms: ns.NodeSelectorTerms.toK8S(),
	}
}

// NodeSelectorTerms represents Kubernetes NodeSelectorTerms
type NodeSelectorTerms []NodeSelectorTerm

// toK8S converts NodeSelectorTerms to Kubernetes client objects
func (nsts NodeSelectorTerms) toK8S() (l []v1.NodeSelectorTerm) {
	if len(nsts) > 0 {
		l = make([]v1.NodeSelectorTerm, 0, len(nsts))
		for _, n := range nsts {
			l = append(l, n.toK8S())
		}
	}
	return
}

// NodeSelectorTerm represents Kubernetes NodeSelectorTerm
type NodeSelectorTerm struct {
	MatchExpressions NodeSelectorRequirements
	MatchFields      NodeSelectorRequirements
}

// toK8S converts NodeSelectorTerm to Kubernetes client object
func (nst *NodeSelectorTerm) toK8S() v1.NodeSelectorTerm {
	return v1.NodeSelectorTerm{
		MatchExpressions: nst.MatchExpressions.toK8S(),
		MatchFields:      nst.MatchFields.toK8S(),
	}
}

// NodeSelectorRequirements represents Kubernetes NodeSelectorRequirements
type NodeSelectorRequirements []NodeSelectorRequirement

// toK8S converts Items to Kubernetes client object
func (nsrs NodeSelectorRequirements) toK8S() (l []v1.NodeSelectorRequirement) {
	if len(nsrs) > 0 {
		l = make([]v1.NodeSelectorRequirement, 0, len(nsrs))
		for _, n := range nsrs {
			l = append(l, n.toK8S())
		}
	}
	return
}

// NodeSelectorRequirement represents Kubernetes NodeSelectorRequirement
type NodeSelectorRequirement struct {
	Key      string
	Operator string
	Values   []string
}

// toK8S converts NodeSelectorRequirement to Kubernetes client object
func (nsr *NodeSelectorRequirement) toK8S() v1.NodeSelectorRequirement {
	return v1.NodeSelectorRequirement{
		Key:      nsr.Key,
		Operator: v1.NodeSelectorOperator(nsr.Operator),
		Values:   nsr.Values,
	}
}

// PodAffinity represents Kubernetes PodAffinity
type PodAffinity struct {
	PreferredDuringSchedulingIgnoredDuringExecution WeightedPodAffinityTerms
	RequiredDuringSchedulingIgnoredDuringExecution  PodAffinityTerms
}

// toK8S converts PodAffinity to Kubernetes client object
func (pa *PodAffinity) toK8S() *v1.PodAffinity {
	return &v1.PodAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: pa.PreferredDuringSchedulingIgnoredDuringExecution.toK8S(),
		RequiredDuringSchedulingIgnoredDuringExecution:  pa.RequiredDuringSchedulingIgnoredDuringExecution.toK8S(),
	}
}

// PodAffinityTerms represents Kubernetes PodAffinityTerms
type PodAffinityTerms []PodAffinityTerm

// toK8S converts PodAffinityTerms to Kubernetes client object
func (pats PodAffinityTerms) toK8S() (l []v1.PodAffinityTerm) {
	if len(pats) > 0 {
		l = make([]v1.PodAffinityTerm, 0, len(pats))
		for _, p := range pats {
			l = append(l, p.toK8S())
		}
	}
	return
}

// PodAffinityTerm represents Kubernetes PodAffinityTerm
type PodAffinityTerm struct {
	LabelSelector map[string]string
	Namespaces    []string
	TopologyKey   string
}

// toK8S converts PodAffinityTerm to Kubernetes client object
func (pat *PodAffinityTerm) toK8S() v1.PodAffinityTerm {
	return v1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{MatchLabels: pat.LabelSelector},
		Namespaces:    pat.Namespaces,
		TopologyKey:   pat.TopologyKey,
	}
}

// WeightedPodAffinityTerms represents Kubernetes WeightedPodAffinityTerms
type WeightedPodAffinityTerms []WeightedPodAffinityTerm

// toK8S converts WeightedPodAffinityTerms to Kubernetes client object
func (wpats WeightedPodAffinityTerms) toK8S() (l []v1.WeightedPodAffinityTerm) {
	if len(wpats) > 0 {
		l = make([]v1.WeightedPodAffinityTerm, 0, len(wpats))
		for _, w := range wpats {
			l = append(l, w.toK8S())
		}
	}
	return
}

// WeightedPodAffinityTerm represents Kubernetes WeightedPodAffinityTerm
type WeightedPodAffinityTerm struct {
	PodAffinityTerm PodAffinityTerm
	Weight          int32
}

// toK8S converts WeightedPodAffinityTerm to Kubernetes client object
func (wpat *WeightedPodAffinityTerm) toK8S() v1.WeightedPodAffinityTerm {
	return v1.WeightedPodAffinityTerm{
		PodAffinityTerm: wpat.PodAffinityTerm.toK8S(),
		Weight:          wpat.Weight,
	}
}

// PodAntiAffinity represents Kubernetes PodAntiAffinity
type PodAntiAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  PodAffinityTerms
	PreferredDuringSchedulingIgnoredDuringExecution WeightedPodAffinityTerms
}

// toK8S converts PodAntiAffinity to Kubernetes client object
func (paa *PodAntiAffinity) toK8S() *v1.PodAntiAffinity {
	return &v1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: paa.PreferredDuringSchedulingIgnoredDuringExecution.toK8S(),
		RequiredDuringSchedulingIgnoredDuringExecution:  paa.RequiredDuringSchedulingIgnoredDuringExecution.toK8S(),
	}
}
