package bee

import (
	"strconv"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

func setInitContainers(clefEnabled, libP2PEnabled, swarmEnabled, initNATPort bool) (inits []statefulset.InitContainer) {
	if clefEnabled || libP2PEnabled || swarmEnabled {
		inits = append(inits, statefulset.InitContainer{
			Name:    "init-keys",
			Image:   "busybox:1.28",
			Command: []string{"sh", "-c", initKeys},
			VolumeMounts: []statefulset.VolumeMount{
				{
					Name:      "data",
					MountPath: "home/bee/.bee",
				},
			},
		})
	}
	if initNATPort {
		inits = append(inits, statefulset.InitContainer{
			Name:    "init-natport",
			Image:   "busybox:1.28",
			Command: []string{"sh", "-c", initP2PFixedPortsCmd},
			VolumeMounts: []statefulset.VolumeMount{
				{
					Name:      "config-file",
					MountPath: "/home/bee",
				},
				{
					Name:      "config",
					MountPath: "/tmp/.bee.yaml",
					SubPath:   ".bee.yaml",
				},
			},
		})
	}

	return
}

func setContainers(name, image, imagePullPolicy, limitCPU, limitMemory, requestCPU, requestMemory string, portAPI, portDebug, portP2P int32, persistenceEnabled, clefEnabled, libP2PEnabled, swarmEnabled bool) []statefulset.Container {
	return []statefulset.Container{{
		Name:            name,
		Image:           image,
		ImagePullPolicy: imagePullPolicy,
		Command:         []string{"bee", "start", "--config=.bee.yaml"},
		Ports: []statefulset.Port{
			{
				Name:          "api",
				ContainerPort: portAPI,
				Protocol:      "TCP",
			},
			{
				Name:          "debug",
				ContainerPort: portDebug,
				Protocol:      "TCP",
			},
			{
				Name:          "p2p",
				ContainerPort: portP2P,
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
			LimitCPU:      limitCPU,
			LimitMemory:   limitMemory,
			RequestCPU:    requestCPU,
			RequestMemory: requestMemory,
		},
		SecurityContext: statefulset.SecurityContext{
			AllowPrivilegeEscalation: false,
			RunAsUser:                999,
		},
		VolumeMounts: setVolumeMounts(persistenceEnabled, clefEnabled, libP2PEnabled, swarmEnabled),
	}}
}

func setVolumeMounts(persistenceEnabled, clefEnabled, libP2PEnabled, swarmEnabled bool) (volumeMounts []statefulset.VolumeMount) {
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
	if clefEnabled {
		volumeMounts = append(volumeMounts, statefulset.VolumeMount{
			Name:      "clef-key",
			MountPath: "home/bee/.bee/keys/clef.key",
			SubPath:   "clef.key",
			ReadOnly:  true,
		})
	}
	if libP2PEnabled {
		volumeMounts = append(volumeMounts, statefulset.VolumeMount{
			Name:      "libp2p-key",
			MountPath: "home/bee/.bee/keys/libp2p.key",
			SubPath:   "libp2p.key",
			ReadOnly:  true,
		})
	}
	if swarmEnabled {
		volumeMounts = append(volumeMounts, statefulset.VolumeMount{
			Name:      "swarm-key",
			MountPath: "home/bee/.bee/keys/swarm.key",
			SubPath:   "swarm.key",
			ReadOnly:  true,
		})
	}

	return
}

func setVolumes(configCM, keysSecret string, persistenceEnabled, clefEnabled, libP2PEnabled, swarmEnabled bool) (volumes []statefulset.Volume) {
	volumes = append(volumes, statefulset.Volume{
		ConfigMap: &statefulset.ConfigMapVolume{
			Name:          "config",
			ConfigMapName: configCM,
		},
	})
	if !persistenceEnabled {
		volumes = append(volumes, statefulset.Volume{
			EmptyDir: &statefulset.EmptyDirVolume{
				Name: "data",
			},
		})
	}
	if clefEnabled {
		volumes = append(volumes, statefulset.Volume{
			Secret: &statefulset.SecretVolume{
				Name:       "clef-key",
				SecretName: keysSecret,
				Items: []statefulset.Item{{
					Key:   "clef",
					Value: "clef.key",
				}},
			},
		})
	}
	if libP2PEnabled {
		volumes = append(volumes, statefulset.Volume{
			Secret: &statefulset.SecretVolume{
				Name:       "libp2p-key",
				SecretName: keysSecret,
				Items: []statefulset.Item{{
					Key:   "libp2p",
					Value: "libp2p.key",
				}},
			},
		})
	}
	if swarmEnabled {
		volumes = append(volumes, statefulset.Volume{
			Secret: &statefulset.SecretVolume{
				Name:       "swarm-key",
				SecretName: keysSecret,
				Items: []statefulset.Item{{
					Key:   "swarm",
					Value: "swarm.key",
				}},
			},
		})
	}

	return
}

func setPersistentVolumeClaims(enabled bool, storageClass, storageRequest string) (pvcs []statefulset.PersistentVolumeClaim) {
	if enabled {
		pvcs = append(pvcs, statefulset.PersistentVolumeClaim{
			Name: "data",
			AccessModes: []statefulset.AccessMode{
				statefulset.AccessMode("ReadWriteOnce"),
			},
			RequestStorage: storageRequest,
			StorageClass:   storageClass,
		})
	}

	return
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
