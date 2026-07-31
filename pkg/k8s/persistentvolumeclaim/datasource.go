package persistentvolumeclaim

import v1 "k8s.io/api/core/v1"

// DataSource represents Kubernetes DataSource
type DataSource struct {
	APIGroup string
	Kind     string
	Name     string
}

// toK8S converts DataSource to Kubernetes client object.
// An unset DataSource must convert to nil, since Kubernetes validates the
// dataSource field whenever it is present and rejects empty kind/name.
func (d *DataSource) toK8S() *v1.TypedLocalObjectReference {
	if d.APIGroup == "" && d.Kind == "" && d.Name == "" {
		return nil
	}

	return &v1.TypedLocalObjectReference{
		APIGroup: &d.APIGroup,
		Kind:     d.Kind,
		Name:     d.Name,
	}
}
