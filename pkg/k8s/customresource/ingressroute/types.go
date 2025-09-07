package ingressroute

import (
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

var (
	_ runtime.Object = (*IngressRoute)(nil)
	_ runtime.Object = (*IngressRouteList)(nil)
)

type IngressRouteSpec struct {
	Routes []Route `json:"routes"`
}

type IngressRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec IngressRouteSpec `json:"spec"`
}

type IngressRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []IngressRoute `json:"items"`
}

type Route struct {
	Kind     string    `json:"kind"`
	Match    string    `json:"match"`
	Services []Service `json:"services"`
}

type Service struct {
	Kind      string      `json:"kind"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Port      interface{} `json:"port"` // TODO: Should be handled more safely ? because we found both string and number in the Cluster
}

// DeepCopyObject implements runtime.Object
func (in *IngressRouteList) DeepCopyObject() runtime.Object {
	out := IngressRouteList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]IngressRoute, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

// DeepCopyObject implements runtime.Object
func (ir *IngressRoute) DeepCopyObject() runtime.Object {
	out := IngressRoute{}
	ir.DeepCopyInto(&out)
	return &out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (ir *IngressRoute) DeepCopyInto(out *IngressRoute) {
	out.TypeMeta = ir.TypeMeta
	out.ObjectMeta = ir.ObjectMeta
	out.Spec = ir.Spec
	copy(out.Spec.Routes, ir.Spec.Routes)
}

func (r *Route) GetHost() string {
	re := regexp.MustCompile(`Host\("([^"]+)"\)`)
	match := re.FindStringSubmatch(r.Match)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}
