package k8s

import (
	"strconv"
	"strings"

	"github.com/ethersphere/beekeeper/pkg/k8s/containers"
	pvc "github.com/ethersphere/beekeeper/pkg/k8s/persistentvolumeclaim"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
)

const (
	configTemplate = `api-addr: {{.APIAddr}}
allow-private-cidrs: {{ .AllowPrivateCIDRs }}
block-time: {{ .BlockTime }}
bootnode: {{.Bootnodes}}
bootnode-mode: {{.BootnodeMode}}
cache-capacity: {{.CacheCapacity}}
clef-signer-enable: {{.ClefSignerEnable}}
clef-signer-endpoint: {{.ClefSignerEndpoint}}
cors-allowed-origins: {{.CORSAllowedOrigins}}
data-dir: {{.DataDir}}
db-open-files-limit: {{.DbOpenFilesLimit}}
db-block-cache-capacity: {{.DbBlockCacheCapacity}}
db-write-buffer-size: {{.DbWriteBufferSize}}
db-disable-seeks-compaction: {{.DbDisableSeeksCompaction}}
debug-api-addr: {{.DebugAPIAddr}}
debug-api-enable: {{.DebugAPIEnable}}
full-node: {{.FullNode}}
mainnet: {{.Mainnet}}
nat-addr: {{.NATAddr}}
network-id: {{.NetworkID}}
p2p-addr: {{.P2PAddr}}
p2p-ws-enable: {{.P2PWSEnable}}
password: {{.Password}}
payment-early-percent: {{.PaymentEarly}}
payment-threshold: {{.PaymentThreshold}}
payment-tolerance-percent: {{.PaymentTolerance}}
postage-stamp-address: {{ .PostageStampAddress }}
postage-stamp-start-block: {{ .PostageContractStartBlock }}
price-oracle-address: {{ .PriceOracleAddress }}
redistribution-address: {{ .RedistributionAddress }}
staking-address: {{ .StakingAddress }}
storage-incentives-enable: {{ .StorageIncentivesEnable }}
resolver-options: {{.ResolverOptions}}
restricted: {{.Restricted}}
token-encryption-key: {{.TokenEncryptionKey}}
admin-password: {{.AdminPassword}}
chequebook-enable: {{.ChequebookEnable}}
swap-enable: {{.SwapEnable}}
swap-endpoint: {{.SwapEndpoint}}
swap-deployment-gas-price: {{.SwapDeploymentGasPrice}}
swap-factory-address: {{.SwapFactoryAddress}}
swap-initial-deposit: {{.SwapInitialDeposit}}
tracing-enable: {{.TracingEnabled}}
tracing-endpoint: {{.TracingEndpoint}}
tracing-service-name: {{.TracingServiceName}}
verbosity: {{.Verbosity}}
welcome-message: {{.WelcomeMessage}}
warmup-time: {{.WarmupTime}}
withdrawal-addresses-whitelist: {{.WithdrawAddress}}
`
)

type setInitContainersOptions struct {
	ClefEnabled         bool
	ClefSecretEnabled   bool
	ClefImage           string
	ClefImagePullPolicy string
	ClefPassword        string
	LibP2PEnabled       bool
	SwarmEnabled        bool
}

func setInitContainers(o setInitContainersOptions) (inits containers.Containers) {
	if o.ClefEnabled {
		inits = append(inits, containers.Container{
			Name:            "init-clef",
			Image:           o.ClefImage,
			ImagePullPolicy: o.ClefImagePullPolicy,
			Command:         []string{"sh", "-c", "/entrypoint.sh init; echo 'clef initialization done';"},
			VolumeMounts: setClefVolumeMounts(setClefVolumeMountsOptions{
				ClefEnabled:       o.ClefEnabled,
				ClefSecretEnabled: o.ClefSecretEnabled,
			}),
		})
	}
	if o.LibP2PEnabled || o.SwarmEnabled {
		inits = append(inits, containers.Container{
			Name:  "init-bee",
			Image: "ethersphere/busybox:1.33",
			Command: []string{"sh", "-c", `mkdir -p /home/bee/.bee/keys;
chown -R 999:999 /home/bee/.bee/keys;
echo 'bee initialization done';`},
			VolumeMounts: containers.VolumeMounts{
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
	Name                   string
	Image                  string
	ImagePullPolicy        string
	PortAPI                int32
	PortDebug              int32
	PortP2P                int32
	PersistenceEnabled     bool
	ResourcesLimitCPU      string
	ResourcesLimitMemory   string
	ResourcesRequestCPU    string
	ResourcesRequestMemory string
	ClefEnabled            bool
	ClefSecretEnabled      bool
	ClefImage              string
	ClefImagePullPolicy    string
	ClefPassword           string
	LibP2PEnabled          bool
	SwarmEnabled           bool
}

func setContainers(o setContainersOptions) (c containers.Containers) {
	c = append(c, containers.Container{
		Name:            "bee",
		Image:           o.Image,
		ImagePullPolicy: o.ImagePullPolicy,
		Command:         []string{"bee", "start", "--config=.bee.yaml"},
		Ports: containers.Ports{
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
		LivenessProbe: containers.Probe{HTTPGet: &containers.HTTPGetProbe{
			InitialDelaySeconds: 5,
			Handler: containers.HTTPGetHandler{
				Path: "/health",
				Port: "debug",
			},
		}},
		ReadinessProbe: containers.Probe{HTTPGet: &containers.HTTPGetProbe{
			InitialDelaySeconds: 5,
			Handler: containers.HTTPGetHandler{
				// Bee node is not ready until it is funded
				// because Beekeeper does funding it needs node to be ready before it is funded
				// if Bee readiness is changed to be ready before funding, path can be set to "/readiness"
				Path: "/health",
				Port: "debug",
			},
		}},
		Resources: containers.Resources{
			Limit: containers.Limit{
				CPU:    o.ResourcesLimitCPU,
				Memory: o.ResourcesLimitMemory,
			},
			Request: containers.Request{
				CPU:    o.ResourcesRequestCPU,
				Memory: o.ResourcesRequestMemory,
			},
		},
		SecurityContext: containers.SecurityContext{
			AllowPrivilegeEscalation: false,
			RunAsUser:                999,
		},
		VolumeMounts: setBeeVolumeMounts(setBeeVolumeMountsOptions{
			LibP2PEnabled: o.LibP2PEnabled,
			SwarmEnabled:  o.SwarmEnabled,
		}),
	})

	if o.ClefEnabled {
		c = append(c, containers.Container{
			Name:            "clef",
			Image:           o.ClefImage,
			ImagePullPolicy: o.ClefImagePullPolicy,
			Command:         []string{"sh", "-c", "/entrypoint.sh run;"},
			Ports: containers.Ports{
				{
					Name:          "api",
					ContainerPort: int32(8550),
					Protocol:      "TCP",
				},
			},
			VolumeMounts: setClefVolumeMounts(setClefVolumeMountsOptions{
				ClefEnabled:       o.ClefEnabled,
				ClefSecretEnabled: o.ClefSecretEnabled,
			}),
		})
	}

	return
}

type setBeeVolumeMountsOptions struct {
	LibP2PEnabled bool
	SwarmEnabled  bool
}

func setBeeVolumeMounts(o setBeeVolumeMountsOptions) (volumeMounts containers.VolumeMounts) {
	volumeMounts = append(volumeMounts, containers.VolumeMount{
		Name:      "config",
		MountPath: "/home/bee/.bee.yaml",
		SubPath:   ".bee.yaml",
		ReadOnly:  true,
	})
	volumeMounts = append(volumeMounts, containers.VolumeMount{
		Name:      "data",
		MountPath: "home/bee/.bee",
	})
	if o.LibP2PEnabled {
		volumeMounts = append(volumeMounts, containers.VolumeMount{
			Name:      "libp2p-key",
			MountPath: "home/bee/.bee/keys/libp2p_v2.key",
			SubPath:   "libp2p_v2.key",
			ReadOnly:  true,
		})
	}
	if o.SwarmEnabled {
		volumeMounts = append(volumeMounts, containers.VolumeMount{
			Name:      "swarm-key",
			MountPath: "home/bee/.bee/keys/swarm.key",
			SubPath:   "swarm.key",
			ReadOnly:  true,
		})
	}

	return
}

type setClefVolumeMountsOptions struct {
	ClefEnabled       bool
	ClefSecretEnabled bool
}

func setClefVolumeMounts(o setClefVolumeMountsOptions) (volumeMounts containers.VolumeMounts) {
	if o.ClefEnabled {
		volumeMounts = append(volumeMounts, containers.VolumeMount{
			Name:      "clef",
			MountPath: "/app/data",
			ReadOnly:  false,
		})
		if o.ClefSecretEnabled {
			volumeMounts = append(volumeMounts, containers.VolumeMount{
				Name:      "clef-key",
				MountPath: "/app/data/keystore/clef.key",
				SubPath:   "clef.key",
				ReadOnly:  true,
			})
			volumeMounts = append(volumeMounts, containers.VolumeMount{
				Name:      "clef-secret",
				MountPath: "/app/data/password",
				SubPath:   "password",
				ReadOnly:  true,
			})
		}
	}

	return
}

type setVolumesOptions struct {
	ConfigCM           string
	KeysSecret         string
	ClefSecret         string
	PersistenceEnabled bool
	ClefEnabled        bool
	ClefSecretEnabled  bool
	LibP2PEnabled      bool
	SwarmEnabled       bool
}

func setVolumes(o setVolumesOptions) (volumes pod.Volumes) {
	volumes = append(volumes, pod.Volume{
		ConfigMap: &pod.ConfigMapVolume{
			Name:          "config",
			ConfigMapName: o.ConfigCM,
		},
	})
	if !o.PersistenceEnabled {
		volumes = append(volumes, pod.Volume{
			EmptyDir: &pod.EmptyDirVolume{
				Name: "data",
			},
		})
	}
	if o.ClefEnabled {
		volumes = append(volumes, pod.Volume{
			EmptyDir: &pod.EmptyDirVolume{
				Name: "clef",
			},
		})
		if o.ClefSecretEnabled {
			volumes = append(volumes, pod.Volume{
				Secret: &pod.SecretVolume{
					Name:       "clef-key",
					SecretName: o.ClefSecret,
					Items: pod.Items{{
						Key:   "key",
						Value: "clef.key",
					}},
				},
			})
			volumes = append(volumes, pod.Volume{
				Secret: &pod.SecretVolume{
					Name:       "clef-secret",
					SecretName: o.ClefSecret,
					Items: pod.Items{{
						Key:   "password",
						Value: "password",
					}},
				},
			})
		}
	}
	if o.LibP2PEnabled {
		volumes = append(volumes, pod.Volume{
			Secret: &pod.SecretVolume{
				Name:       "libp2p-key",
				SecretName: o.KeysSecret,
				Items: pod.Items{{
					Key:   "libp2p",
					Value: "libp2p_v2.key",
				}},
			},
		})
	}
	if o.SwarmEnabled {
		volumes = append(volumes, pod.Volume{
			Secret: &pod.SecretVolume{
				Name:       "swarm-key",
				SecretName: o.KeysSecret,
				Items: pod.Items{{
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

func setPersistentVolumeClaims(o setPersistentVolumeClaimsOptions) (pvcs pvc.PersistentVolumeClaims) {
	if o.Enabled {
		pvcs = append(pvcs, pvc.PersistentVolumeClaim{
			Name: "data",
			Spec: pvc.PersistentVolumeClaimSpec{
				AccessModes: pvc.AccessModes{
					pvc.AccessMode("ReadWriteOnce"),
				},
				RequestStorage: o.StorageRequest,
				StorageClass:   o.StorageClass,
			},
		})
	}

	return
}

type setBeeNodePortOptions struct {
	AppProtocol string
	Name        string
	Protocol    string
	TargetPort  string
	Port        int32
	NodePort    int32
}

func setBeeNodePort(o setBeeNodePortOptions) (ports service.Ports) {
	if o.NodePort > 0 {
		return service.Ports{{
			AppProtocol: "TCP",
			Name:        "p2p",
			Protocol:    "TCP",
			Port:        o.Port,
			TargetPort:  "p2p",
			Nodeport:    o.NodePort,
		}}
	}
	return service.Ports{{
		AppProtocol: "TCP",
		Name:        "p2p",
		Protocol:    "TCP",
		Port:        o.Port,
		TargetPort:  "p2p",
	}}
}

func parsePort(port string) (int32, error) {
	p, err := strconv.ParseInt(strings.Split(port, ":")[1], 10, 32)
	return int32(p), err
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
