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

func (v Volume) toK8S() v1.Volume {
	if v.EmptyDir != nil {
		return v.EmptyDir.toK8S()
	} else if v.ConfigMap != nil {
		return v.ConfigMap.toK8S()
	} else if v.Secret != nil {
		return v.Secret.toK8S()
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

// EmptyDirVolume ...
type EmptyDirVolume struct {
	Name      string
	Medium    string
	SizeLimit string
}

func (ed EmptyDirVolume) toK8S() v1.Volume {
	return v1.Volume{
		Name: ed.Name,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{
				Medium: v1.StorageMedium(ed.Medium),
				SizeLimit: func() *resource.Quantity {
					if len(ed.SizeLimit) > 0 {
						r := resource.MustParse(ed.SizeLimit)
						return &r
					}
					return nil
				}(),
			},
		},
	}
}

// ConfigMapVolume ...
type ConfigMapVolume struct {
	Name          string
	ConfigMapName string
	DefaultMode   int32
	Items         []Item
	Optional      bool
}

func (cm ConfigMapVolume) toK8S() v1.Volume {
	return v1.Volume{
		Name: cm.Name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: cm.ConfigMapName},
				DefaultMode:          &cm.DefaultMode,
				Items:                itemsToK8S(cm.Items),
				Optional:             &cm.Optional,
			},
		},
	}
}

// SecretVolume ...
type SecretVolume struct {
	Name        string
	SecretName  string
	DefaultMode int32
	Items       []Item
	Optional    bool
}

func (s SecretVolume) toK8S() v1.Volume {
	return v1.Volume{
		Name: s.Name,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  s.SecretName,
				DefaultMode: &s.DefaultMode,
				Items:       itemsToK8S(s.Items),
				Optional:    &s.Optional,
			},
		},
	}
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
