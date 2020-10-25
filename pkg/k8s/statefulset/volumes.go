package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Volume ...
type Volume struct {
	ConfigMap *ConfigMapVolume
	EmptyDir  *EmptyDirVolume
	Secret    *SecretVolume
}

// EmptyDirVolume ...
type EmptyDirVolume struct {
	Name      string
	Medium    string
	SizeLimit string
}

// ConfigMapVolume ...
type ConfigMapVolume struct {
	Name          string
	ConfigMapName string
	DefaultMode   int32
	Items         []Item
	Optional      bool
}

// SecretVolume ...
type SecretVolume struct {
	Name        string
	SecretName  string
	DefaultMode int32
	Items       []Item
	Optional    bool
}

func (v Volume) toK8S() v1.Volume {
	if v.EmptyDir != nil {
		return v1.Volume{
			Name: v.EmptyDir.Name,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: v1.StorageMedium(v.EmptyDir.Medium),
					SizeLimit: func() *resource.Quantity {
						r := resource.MustParse(v.EmptyDir.SizeLimit)
						return &r
					}(),
				},
			},
		}
	} else if v.ConfigMap != nil {
		return v1.Volume{
			Name: v.ConfigMap.Name,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: v.ConfigMap.ConfigMapName},
					DefaultMode:          &v.ConfigMap.DefaultMode,
					Items:                itemsToK8S(v.ConfigMap.Items),
					Optional:             &v.ConfigMap.Optional,
				},
			},
		}
	} else if v.Secret != nil {
		return v1.Volume{
			Name: v.Secret.Name,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName:  v.Secret.SecretName,
					DefaultMode: &v.Secret.DefaultMode,
					Items:       itemsToK8S(v.Secret.Items),
					Optional:    &v.ConfigMap.Optional,
				},
			},
		}
	} else {
		return v1.Volume{}
	}
}

func volumesToK8S(volumes []Volume) (vs []v1.Volume) {
	for _, volume := range volumes {
		vs = append(vs, volume.toK8S())
	}
	return
}

// Item ...
type Item struct {
	Key   string
	Value string
}

func (i Item) toK8S() v1.KeyToPath {
	return v1.KeyToPath{
		Key:  i.Key,
		Path: i.Value,
	}
}

func itemsToK8S(items []Item) (is []v1.KeyToPath) {
	for _, item := range items {
		is = append(is, item.toK8S())
	}
	return
}
