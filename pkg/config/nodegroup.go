package config

import (
	"reflect"

	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// NodeGroup represents node group configuration
type NodeGroup struct {
	// parent to inherit settings from
	*Inherit `yaml:",inline"`
	// node group configuration
	Annotations               *map[string]string `yaml:"annotations"`
	Image                     *string            `yaml:"image"`
	ImagePullPolicy           *string            `yaml:"image-pull-policy"`
	ImagePullSecrets          *[]string          `yaml:"image-pull-secrets"`
	IngressAnnotations        *map[string]string `yaml:"ingress-annotations"`
	IngressClass              *string            `yaml:"ingress-class"`
	Labels                    *map[string]string `yaml:"labels"`
	NodeSelector              *map[string]string `yaml:"node-selector"`
	PersistenceEnabled        *bool              `yaml:"persistence-enabled"`
	PersistenceStorageClass   *string            `yaml:"persistence-storage-class"`
	PersistenceStorageRequest *string            `yaml:"persistence-storage-request"`
	PodManagementPolicy       *string            `yaml:"pod-management-policy"`
	ResourcesLimitCPU         *string            `yaml:"resources-limit-cpu"`
	ResourcesLimitMemory      *string            `yaml:"resources-limit-memory"`
	ResourcesRequestCPU       *string            `yaml:"resources-request-cpu"`
	ResourcesRequestMemory    *string            `yaml:"resources-request-memory"`
	RestartPolicy             *string            `yaml:"restart-policy"`
	UpdateStrategy            *string            `yaml:"update-strategy"`
}

func (b NodeGroup) GetParentName() string {
	if b.Inherit != nil {
		return b.ParentName
	}
	return ""
}

// Export exports NodeGroup to orchestration.NodeGroupOptions
func (n *NodeGroup) Export() (o orchestration.NodeGroupOptions) {
	localVal := reflect.ValueOf(n).Elem()
	localType := reflect.TypeOf(n).Elem()
	remoteVal := reflect.ValueOf(&o).Elem()

	for i := 0; i < localVal.NumField(); i++ {
		localField := localVal.Field(i)
		if localField.IsValid() && !localField.IsNil() {
			localFieldVal := localVal.Field(i).Elem()
			localFieldName := localType.Field(i).Name

			remoteFieldVal := remoteVal.FieldByName(localFieldName)
			if remoteFieldVal.IsValid() && remoteFieldVal.Type() == localFieldVal.Type() {
				remoteFieldVal.Set(localFieldVal)
			}
		}
	}

	return remoteVal.Interface().(orchestration.NodeGroupOptions)
}
