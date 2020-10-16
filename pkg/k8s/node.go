package k8s

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

// NodeStartOptions ...
type NodeStartOptions struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Config      string
}

// NodeStart ...
func (c Client) NodeStart(ctx context.Context, o NodeStartOptions) (err error) {
	// configuration
	if err = c.ConfigMap.Set(ctx, o.Name, o.Namespace, configmap.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Data: map[string]string{
			".bee.yaml": o.Config,
		},
	}); err != nil {
		return fmt.Errorf("set ConfigMap: %s", err)
	}

	if err := c.Secret.Set(ctx, fmt.Sprintf("%s-libp2p", o.Name), o.Namespace, secret.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		StringData: map[string]string{
			"libp2pKeys": `bee-0: {"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}
bee-1: {"address":"03348ecf3adae1d05dc16e475a83c94e49e28a4d3c7db5eccbf5ca4ea7f688ddcdfe88acbebef2037c68030b1a0a367a561333e5c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470","crypto":{"cipher":"aes-128-ctr","ciphertext":"0d0ff25e9b03292e622c5a09ec00c2acb7ff5882f02dd2f00a26ac6d3292a434","cipherparams":{"iv":"cd4082caf63320b306fe885796ba224f"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a4d63d56c539eb3eff2a235090127486722fa2c836cf735d50d673b730cebc3f"},"mac":"aad40da9c1e742e2b01bb8f76ba99ace97ccb0539cea40e31eb6b9bb64a3f36a"},"version":3}`,
		},
	}); err != nil {
		return fmt.Errorf("set Secret: %s", err)
	}

	// services
	if err := c.ServiceAccount.Set(ctx, o.Name, o.Namespace, serviceaccount.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
	}); err != nil {
		return fmt.Errorf("set ServiceAccount %s", err)
	}

	if err := c.Service.Set(ctx, o.Name, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Ports: []service.Port{{
			Name:       "http",
			Protocol:   "TCP",
			Port:       80,
			TargetPort: "api",
		}},
		Selector: map[string]string{
			"app.kubernetes.io/instance":   "bee",
			"app.kubernetes.io/name":       "bee",
			"app.kubernetes.io/managed-by": "beekeeper",
		},
		Type: "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set Service %s", err)
	}

	if err := c.Service.Set(ctx, fmt.Sprintf("%s-headless", o.Name), o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Ports: []service.Port{
			{
				Name:       "api",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: "api",
			},
			{
				Name:       "p2p",
				Protocol:   "TCP",
				Port:       7070,
				TargetPort: "p2p",
			},
			{
				Name:       "debug",
				Protocol:   "TCP",
				Port:       6060,
				TargetPort: "debug",
			},
		},
		Selector: map[string]string{
			"app.kubernetes.io/instance":   "bee",
			"app.kubernetes.io/name":       "bee",
			"app.kubernetes.io/managed-by": "beekeeper",
		},
		Type: "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set Service %s", err)
	}

	// ingress
	if err := c.Ingress.Set(ctx, o.Name, o.Namespace, ingress.Options{
		Annotations: map[string]string{
			"createdBy":                                          "beekeeper",
			"kubernetes.io/ingress.class":                        "nginx-internal",
			"nginx.ingress.kubernetes.io/affinity":               "cookie",
			"nginx.ingress.kubernetes.io/affinity-mode":          "persistent",
			"nginx.ingress.kubernetes.io/proxy-body-size":        "0",
			"nginx.ingress.kubernetes.io/proxy-read-timeout":     "7200",
			"nginx.ingress.kubernetes.io/proxy-send-timeout":     "7200",
			"nginx.ingress.kubernetes.io/session-cookie-max-age": "86400",
			"nginx.ingress.kubernetes.io/session-cookie-name":    "SWARMGATEWAY",
			"nginx.ingress.kubernetes.io/session-cookie-path":    "default",
			"nginx.ingress.kubernetes.io/ssl-redirect":           "true",
		},
		Labels:      o.Labels,
		Class:       "nginx-internal",
		Host:        "bee.beekeeper.staging.internal",
		ServiceName: o.Name,
		ServicePort: "http",
		Path:        "/",
	}); err != nil {
		return fmt.Errorf("set Ingress %s", err)
	}

	// statefulset
	if err := c.StatefulSet.Set(ctx, fmt.Sprintf("%s-0", o.Name), o.Namespace, statefulset.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Replicas:    1,
		Selector: map[string]string{
			"app.kubernetes.io/name":       "bee",
			"app.kubernetes.io/managed-by": "beekeeper",
		},
		RestartPolicy:      "Always",
		ServiceAccountName: o.Name,
		ServiceName:        fmt.Sprintf("%s-headless", o.Name),
		NodeSelector: map[string]string{
			"node-group": "bee-staging",
		},
		PodManagementPolicy: "OrderedReady",
		PodSecurityContext: statefulset.PodSecurityContext{
			FSGroup: 999,
		},
		UpdateStrategy: statefulset.UpdateStrategy{
			Type: "OnDelete",
		},
		Volumes: []statefulset.Volume{
			{ConfigMap: &statefulset.ConfigMapVolume{
				Name:          "config",
				ConfigMapName: o.Name,
				DefaultMode:   420,
			}},
			{EmptyDir: &statefulset.EmptyDirVolume{
				Name: "data",
			}},
			{Secret: &statefulset.SecretVolume{
				Name:        fmt.Sprintf("%s-libp2p", o.Name),
				SecretName:  fmt.Sprintf("%s-libp2p", o.Name),
				DefaultMode: 420,
				Items: []statefulset.Item{{
					Key:   "libp2pKeys",
					Value: "libp2p.map",
				}},
			}},
		},
		InitContainers: []statefulset.InitContainer{{
			Name:    "init-libp2p",
			Image:   "busybox:1.28",
			Command: []string{"sh", "-c", `export INDEX=$(echo $(hostname) | rev | cut -d'-' -f 1 | rev); mkdir -p /home/bee/.bee/keys; chown -R 999:999 /home/bee/.bee/keys; export KEY=$(cat /tmp/bee/libp2p.map | grep bee-${INDEX}: | cut -d' ' -f2); if [ -z "${KEY}" ]; then exit 0; fi; printf '%s' "${KEY}" > /home/bee/.bee/keys/libp2p.key; echo 'node initialization done';`},
			VolumeMounts: []statefulset.VolumeMount{
				{Name: "bee-libp2p", MountPath: "/tmp/bee"},
				{Name: "data", MountPath: "home/bee/.bee"},
			},
		}},
		Containers: []statefulset.Container{{
			Name:            o.Name,
			Image:           "ethersphere/bee:latest",
			ImagePullPolicy: "Always",
			Command:         []string{"bee", "start", "--config=.bee.yaml"},
			Ports: []statefulset.Port{
				{
					Name:          "api",
					ContainerPort: 8080,
					Protocol:      "TCP",
				},
				{
					Name:          "p2p",
					ContainerPort: 7070,
					Protocol:      "TCP",
				},
				{
					Name:          "debug",
					ContainerPort: 6060,
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
				LimitCPU:      "1",
				LimitMemory:   "2Gi",
				RequestCPU:    "750m",
				RequestMemory: "1Gi",
			},
			SecurityContext: statefulset.SecurityContext{
				AllowPrivilegeEscalation: false,
				RunAsUser:                999,
			},
			VolumeMounts: []statefulset.VolumeMount{
				{Name: "config", MountPath: "/home/bee/.bee.yaml", SubPath: ".bee.yaml", ReadOnly: true},
				{Name: "data", MountPath: "home/bee/.bee"},
			},
		}},
	}); err != nil {
		return fmt.Errorf("set StatefulSet %s", err)
	}

	fmt.Println("Node started")
	return
}
