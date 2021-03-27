package config

type NodeGroupProfile struct {
	Profile   `yaml:",inline"`
	NodeGroup `yaml:",inline"`
}

type NodeGroup struct {
	Annotations *map[string]string `yaml:"annotations"`
	ClefImage   *struct {
		Name       string `yaml:"name"`
		Tag        string `yaml:"tag"`
		PullPolicy string `yaml:"pull-policy"`
	} `yaml:"clef-image"`
	Image *struct {
		Name       string `yaml:"name"`
		Tag        string `yaml:"tag"`
		PullPolicy string `yaml:"pull-policy"`
	} `yaml:"image"`
	IngressAnnotations      *map[string]string `yaml:"ingress-annotations"`
	IngressDebugAnnotations *map[string]string `yaml:"ingress-debug-annotations"`
	Labels                  *map[string]string `yaml:"labels"`
	NodeSelector            *map[string]string `yaml:"node-selector"`
	Persistence             *struct {
		Enabled        bool   `yaml:"enabled"`
		StorageClass   string `yaml:"storage-class"`
		StorageRequest string `yaml:"storage-request"`
	} `yaml:"persistence"`
	PodManagementPolicy *string `yaml:"pod-management-policy"`
	Resources           *struct {
		Limit struct {
			CPU    string `yaml:"cpu"`
			Memory string `yaml:"memory"`
		} `yaml:"limit"`
		Request struct {
			CPU    string `yaml:"cpu"`
			Memory string `yaml:"memory"`
		} `yaml:"request"`
	} `yaml:"resources"`
	RestartPolicy  *string `yaml:"restart-policy"`
	UpdateStrategy *string `yaml:"update-strategy"`
}
