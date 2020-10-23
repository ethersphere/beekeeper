package bee

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

type setInitContainersOptions struct {
	ClefEnabled         bool
	ClefImage           string
	ClefImagePullPolicy string
	ClefPassword        string
	LibP2PEnabled       bool
	SwarmEnabled        bool
}

func setInitContainers(o setInitContainersOptions) (inits []statefulset.InitContainer) {
	if o.ClefEnabled {
		inits = append(inits, statefulset.InitContainer{
			Name:            "init-clef",
			Image:           o.ClefImage,
			ImagePullPolicy: o.ClefImagePullPolicy,
			Command:         []string{"sh", "-c", fmt.Sprintf("/entrypoint.sh init %s; echo 'clef initialization done';", o.ClefPassword)},
			VolumeMounts: setClefVolumeMounts(setClefVolumeMountsOptions{
				ClefEnabled: o.ClefEnabled,
			}),
		})
	}
	if o.LibP2PEnabled || o.SwarmEnabled {
		inits = append(inits, statefulset.InitContainer{
			Name:  "init-bee",
			Image: "busybox:1.28",
			Command: []string{"sh", "-c", `mkdir -p /home/bee/.bee/keys;
chown -R 999:999 /home/bee/.bee/keys;
echo 'bee initialization done';`},
			VolumeMounts: []statefulset.VolumeMount{
				{
					Name:      "data",
					MountPath: "home/bee/.bee",
				},
			},
		})
	}

	return
}

type setContainersOptions struct {
	Name                string
	Image               string
	ImagePullPolicy     string
	LimitCPU            string
	LimitMemory         string
	RequestCPU          string
	RequestMemory       string
	PortAPI             int32
	PortDebug           int32
	PortP2P             int32
	PersistenceEnabled  bool
	ClefEnabled         bool
	ClefImage           string
	ClefImagePullPolicy string
	ClefPassword        string
	LibP2PEnabled       bool
	SwarmEnabled        bool
}

func setContainers(o setContainersOptions) (containers []statefulset.Container) {
	containers = append(containers, statefulset.Container{
		Name:            o.Name,
		Image:           o.Image,
		ImagePullPolicy: o.ImagePullPolicy,
		Command:         []string{"bee", "start", "--config=.bee.yaml"},
		Ports: []statefulset.Port{
			{
				Name:          "api",
				ContainerPort: o.PortAPI,
				Protocol:      "TCP",
			},
			{
				Name:          "debug",
				ContainerPort: o.PortDebug,
				Protocol:      "TCP",
			},
			{
				Name:          "p2p",
				ContainerPort: o.PortP2P,
				Protocol:      "TCP",
			},
		},
		LivenessProbe: statefulset.Probe{
			Path: "/health",
			Port: "debug",
		},
		ReadinessProbe: statefulset.Probe{
			Path: "/readiness",
			Port: "debug",
		},
		Resources: statefulset.Resources{
			LimitCPU:      o.LimitCPU,
			LimitMemory:   o.LimitMemory,
			RequestCPU:    o.RequestCPU,
			RequestMemory: o.RequestMemory,
		},
		SecurityContext: statefulset.SecurityContext{
			AllowPrivilegeEscalation: false,
			RunAsUser:                999,
		},
		VolumeMounts: setBeeVolumeMounts(setBeeVolumeMountsOptions{
			LibP2PEnabled: o.LibP2PEnabled,
			SwarmEnabled:  o.SwarmEnabled,
		}),
	})

	if o.ClefEnabled {
		containers = append(containers, statefulset.Container{
			Name:            "clef",
			Image:           o.ClefImage,
			ImagePullPolicy: o.ClefImagePullPolicy,
			Command:         []string{"sh", "-c", fmt.Sprintf("/entrypoint.sh run %s;", o.ClefPassword)},
			Ports: []statefulset.Port{
				{
					Name:          "api",
					ContainerPort: int32(8550),
					Protocol:      "TCP",
				},
			},
			VolumeMounts: setClefVolumeMounts(setClefVolumeMountsOptions{
				ClefEnabled: o.ClefEnabled,
			}),
		})
	}

	return
}

type setBeeVolumeMountsOptions struct {
	LibP2PEnabled bool
	SwarmEnabled  bool
}

func setBeeVolumeMounts(o setBeeVolumeMountsOptions) (volumeMounts []statefulset.VolumeMount) {
	volumeMounts = append(volumeMounts, statefulset.VolumeMount{
		Name:      "config",
		MountPath: "/home/bee/.bee.yaml",
		SubPath:   ".bee.yaml",
		ReadOnly:  true,
	})
	volumeMounts = append(volumeMounts, statefulset.VolumeMount{
		Name:      "data",
		MountPath: "home/bee/.bee",
	})
	if o.LibP2PEnabled {
		volumeMounts = append(volumeMounts, statefulset.VolumeMount{
			Name:      "libp2p-key",
			MountPath: "home/bee/.bee/keys/libp2p.key",
			SubPath:   "libp2p.key",
			ReadOnly:  true,
		})
	}
	if o.SwarmEnabled {
		volumeMounts = append(volumeMounts, statefulset.VolumeMount{
			Name:      "swarm-key",
			MountPath: "home/bee/.bee/keys/swarm.key",
			SubPath:   "swarm.key",
			ReadOnly:  true,
		})
	}

	return
}

type setClefVolumeMountsOptions struct {
	ClefEnabled bool
}

func setClefVolumeMounts(o setClefVolumeMountsOptions) (volumeMounts []statefulset.VolumeMount) {
	if o.ClefEnabled {
		volumeMounts = append(volumeMounts, statefulset.VolumeMount{
			Name:      "clef-key",
			MountPath: "/root/.clef/keys/clef.key",
			SubPath:   "clef.key",
			ReadOnly:  true,
		})
	}

	return
}

type setVolumesOptions struct {
	ConfigCM           string
	KeysSecret         string
	ClefKeySecret      string
	PersistenceEnabled bool
	ClefEnabled        bool
	LibP2PEnabled      bool
	SwarmEnabled       bool
}

func setVolumes(o setVolumesOptions) (volumes []statefulset.Volume) {
	volumes = append(volumes, statefulset.Volume{
		ConfigMap: &statefulset.ConfigMapVolume{
			Name:          "config",
			ConfigMapName: o.ConfigCM,
		},
	})
	if !o.PersistenceEnabled {
		volumes = append(volumes, statefulset.Volume{
			EmptyDir: &statefulset.EmptyDirVolume{
				Name: "data",
			},
		})
	}
	if o.ClefEnabled {
		volumes = append(volumes, statefulset.Volume{
			Secret: &statefulset.SecretVolume{
				Name:       "clef-key",
				SecretName: o.ClefKeySecret,
				Items: []statefulset.Item{{
					Key:   "clef",
					Value: "clef.key",
				}},
			},
		})
	}
	if o.LibP2PEnabled {
		volumes = append(volumes, statefulset.Volume{
			Secret: &statefulset.SecretVolume{
				Name:       "libp2p-key",
				SecretName: o.KeysSecret,
				Items: []statefulset.Item{{
					Key:   "libp2p",
					Value: "libp2p.key",
				}},
			},
		})
	}
	if o.SwarmEnabled {
		volumes = append(volumes, statefulset.Volume{
			Secret: &statefulset.SecretVolume{
				Name:       "swarm-key",
				SecretName: o.KeysSecret,
				Items: []statefulset.Item{{
					Key:   "swarm",
					Value: "swarm.key",
				}},
			},
		})
	}

	return
}

type setPersistentVolumeClaimsOptions struct {
	Enabled        bool
	StorageClass   string
	StorageRequest string
}

func setPersistentVolumeClaims(o setPersistentVolumeClaimsOptions) (pvcs []statefulset.PersistentVolumeClaim) {
	if o.Enabled {
		pvcs = append(pvcs, statefulset.PersistentVolumeClaim{
			Name: "data",
			AccessModes: []statefulset.AccessMode{
				statefulset.AccessMode("ReadWriteOnce"),
			},
			RequestStorage: o.StorageRequest,
			StorageClass:   o.StorageClass,
		})
	}

	return
}

type setBeeNodePortOptions struct {
	Name       string
	Protocol   string
	TargetPort string
	Port       int32
	NodePort   int32
}

func setBeeNodePort(o setBeeNodePortOptions) (ports []service.Port) {
	if o.NodePort > 0 {
		return []service.Port{
			{
				Name:       "p2p",
				Protocol:   "TCP",
				Port:       o.Port,
				TargetPort: "p2p",
				Nodeport:   o.NodePort,
			},
		}
	}
	return []service.Port{
		{
			Name:       "p2p",
			Protocol:   "TCP",
			Port:       o.Port,
			TargetPort: "p2p",
		},
	}
}

func mergeMaps(a, b map[string]string) map[string]string {
	m := map[string]string{}
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}

	return m
}

func parsePort(port string) (int32, error) {
	p, err := strconv.ParseInt(strings.Split(port, ":")[1], 10, 32)
	return int32(p), err
}
