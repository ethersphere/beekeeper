package config

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
)

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

// TODO: with reflex
func (n *NodeGroup) Export() bee.NodeGroupOptions {
	return bee.NodeGroupOptions{
		// Annotations:               *n.Annotations,
		ClefImage:                 fmt.Sprintf("%s:%s", n.ClefImage.Name, n.ClefImage.Tag),
		ClefImagePullPolicy:       n.ClefImage.PullPolicy,
		BeeConfig:                 nil,
		Image:                     fmt.Sprintf("%s:%s", n.Image.Name, n.Image.Tag),
		IngressAnnotations:        *n.IngressAnnotations,
		IngressDebugAnnotations:   *n.IngressDebugAnnotations,
		Labels:                    *n.Labels,
		LimitCPU:                  n.Resources.Limit.CPU,
		LimitMemory:               n.Resources.Limit.Memory,
		NodeSelector:              *n.NodeSelector,
		PersistenceEnabled:        n.Persistence.Enabled,
		PersistenceStorageClass:   n.Persistence.StorageClass,
		PersistanceStorageRequest: n.Persistence.StorageRequest,
		PodManagementPolicy:       *n.PodManagementPolicy,
		RestartPolicy:             *n.RestartPolicy,
		RequestCPU:                n.Resources.Request.CPU,
		RequestMemory:             n.Resources.Request.Memory,
		UpdateStrategy:            *n.UpdateStrategy,
	}
}
