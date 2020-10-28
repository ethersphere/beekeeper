package persistentvolumeclaim

import v1 "k8s.io/api/core/v1"

// DataSource represents Kubernetes DataSource
type DataSource struct {
	APIGroup string
	Kind     string
	Name     string
}

// toK8S converts DataSource to Kuberntes client object
func (d DataSource) toK8S() *v1.TypedLocalObjectReference {
	return &v1.TypedLocalObjectReference{
		APIGroup: &d.APIGroup,
		Kind:     d.Kind,
		Name:     d.Name,
	}
}
