package pod

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Volumes represents Kubernetes Volumes
type Volumes []Volume

// toK8S converts Volumes to Kuberntes client objects
func (vs Volumes) toK8S() (l []v1.Volume) {
	l = make([]v1.Volume, 0, len(vs))

	for _, v := range vs {
		l = append(l, v.toK8S())
	}

	return
}

// Volume represents Kubernetes Volume
type Volume struct {
	ConfigMap *ConfigMapVolume
	EmptyDir  *EmptyDirVolume
	Secret    *SecretVolume
}

// toK8S converts Volume to Kuberntes client object
func (v *Volume) toK8S() v1.Volume {
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

// EmptyDirVolume represents Kubernetes EmptyDir Volume
type EmptyDirVolume struct {
	Name      string
	Medium    string
	SizeLimit string
}

// toK8S converts EmptyDirVolume to Kuberntes client object
func (ed *EmptyDirVolume) toK8S() v1.Volume {
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

// ConfigMapVolume represents Kubernetes ConfigMap Volume
type ConfigMapVolume struct {
	Name          string
	ConfigMapName string
	DefaultMode   int32
	Items         Items
	Optional      bool
}

// toK8S converts ConfigMapVolume to Kuberntes client object
func (cm *ConfigMapVolume) toK8S() v1.Volume {
	return v1.Volume{
		Name: cm.Name,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: cm.ConfigMapName},
				DefaultMode:          &cm.DefaultMode,
				Items:                cm.Items.toK8S(),
				Optional:             &cm.Optional,
			},
		},
	}
}

// SecretVolume represents Kubernetes Secret Volume
type SecretVolume struct {
	Name        string
	SecretName  string
	DefaultMode int32
	Items       Items
	Optional    bool
}

// toK8S converts SecretVolume to Kuberntes client object
func (s *SecretVolume) toK8S() v1.Volume {
	return v1.Volume{
		Name: s.Name,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  s.SecretName,
				DefaultMode: &s.DefaultMode,
				Items:       s.Items.toK8S(),
				Optional:    &s.Optional,
			},
		},
	}
}

// Items represents Kubernetes Volume Items
type Items []Item

// toK8S converts Items to Kuberntes client object
func (is Items) toK8S() (l []v1.KeyToPath) {
	l = make([]v1.KeyToPath, 0, len(is))

	for _, i := range is {
		l = append(l, i.toK8S())
	}

	return
}

// Item represents Kubernetes Volume Item
type Item struct {
	Key   string
	Value string
}

// toK8S converts Item to Kuberntes client object
func (i *Item) toK8S() v1.KeyToPath {
	return v1.KeyToPath{
		Key:  i.Key,
		Path: i.Value,
	}
}
