package orchestration

// NodeGroupOptions represents node group options
type NodeGroupOptions struct {
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	BeeConfig                 *Config
	Image                     string
	ImagePullPolicy           string
	ImagePullSecrets          []string
	IngressAnnotations        map[string]string
	IngressClass              string
	IngressDebugAnnotations   map[string]string
	IngressDebugClass         string
	Labels                    map[string]string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistenceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	ResourcesLimitCPU         string
	ResourcesLimitMemory      string
	ResourcesRequestCPU       string
	ResourcesRequestMemory    string
	UpdateStrategy            string
}
