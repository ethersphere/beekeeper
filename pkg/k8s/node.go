package k8s

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

const (
	libp2pKeys = `bee-0: {"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}
bee-1: {"address":"03348ecf3adae1d05dc16e475a83c94e49e28a4d3c7db5eccbf5ca4ea7f688ddcdfe88acbebef2037c68030b1a0a367a561333e5c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470","crypto":{"cipher":"aes-128-ctr","ciphertext":"0d0ff25e9b03292e622c5a09ec00c2acb7ff5882f02dd2f00a26ac6d3292a434","cipherparams":{"iv":"cd4082caf63320b306fe885796ba224f"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a4d63d56c539eb3eff2a235090127486722fa2c836cf735d50d673b730cebc3f"},"mac":"aad40da9c1e742e2b01bb8f76ba99ace97ccb0539cea40e31eb6b9bb64a3f36a"},"version":3}`
	initCmd  = `export INDEX=$(echo $(hostname) | rev | cut -d'-' -f 1 | rev); mkdir -p /home/bee/.bee/keys; chown -R 999:999 /home/bee/.bee/keys; export KEY=$(cat /tmp/bee/libp2p.map | grep bee-${INDEX}: | cut -d' ' -f2); if [ -z "${KEY}" ]; then exit 0; fi; printf '%s' "${KEY}" > /home/bee/.bee/keys/libp2p.key; echo 'node initialization done';`
	portHTTP = 80
)

// NodeStartOptions ...
type NodeStartOptions struct {
	// Bee configuration
	Config Config
	// Kubernetes configuration
	Name                    string
	Namespace               string
	Annotations             map[string]string
	Labels                  map[string]string
	LimitCPU                string
	LimitMemory             string
	Image                   string
	ImagePullPolicy         string
	IngressAnnotations      map[string]string
	IngressHost             string
	IngressDebugAnnotations map[string]string
	IngressDebugHost        string
	NodeSelector            map[string]string
	PodManagementPolicy     string
	RestartPolicy           string
	RequestCPU              string
	RequestMemory           string
	Selector                map[string]string
	UpdateStrategy          string
}

// NodeStart ...
func (c Client) NodeStart(ctx context.Context, o NodeStartOptions) (err error) {
	// bee configuration
	var config bytes.Buffer
	if err := template.Must(template.New("").Parse(configTemplate)).Execute(&config, o.Config); err != nil {
		return err
	}

	configCM := o.Name
	if err = c.ConfigMap.Set(ctx, configCM, o.Namespace, configmap.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Data: map[string]string{
			".bee.yaml": config.String(),
		},
	}); err != nil {
		return fmt.Errorf("set configmap in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("configmap %s is set in namespace %s\n", configCM, o.Namespace)

	// secret with libp2p keys
	libp2pSecret := fmt.Sprintf("%s-libp2p", o.Name)
	if err := c.Secret.Set(ctx, libp2pSecret, o.Namespace, secret.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		StringData: map[string]string{
			"libp2pKeys": libp2pKeys,
		},
	}); err != nil {
		return fmt.Errorf("set secret in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("secret %s is set in namespace %s\n", libp2pSecret, o.Namespace)

	// service account
	svcAccount := o.Name
	if err := c.ServiceAccount.Set(ctx, svcAccount, o.Namespace, serviceaccount.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
	}); err != nil {
		return fmt.Errorf("set serviceaccount in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("serviceaccount %s is set in namespace %s\n", svcAccount, o.Namespace)

	// api service
	portAPI, err := parsePort(o.Config.APIAddr)
	if err != nil {
		return fmt.Errorf("parsing API port from config: %s", err)
	}

	apiSvc := fmt.Sprintf("%s-api", o.Name)
	if err := c.Service.Set(ctx, apiSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Ports: []service.Port{
			{
				Name:       "api",
				Protocol:   "TCP",
				Port:       portHTTP,
				TargetPort: "api",
			},
		},
		Selector: o.Selector,
		Type:     "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", apiSvc, o.Namespace)

	// api service's ingress
	apiIn := fmt.Sprintf("%s-api", o.Name)
	if err := c.Ingress.Set(ctx, apiIn, o.Namespace, ingress.Options{
		Annotations: mergeMaps(o.Annotations, o.IngressAnnotations),
		Labels:      o.Labels,
		Host:        o.IngressHost,
		ServiceName: apiSvc,
		ServicePort: "api",
		Path:        "/",
	}); err != nil {
		return fmt.Errorf("set ingress in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("ingress %s is set in namespace %s\n", apiIn, o.Namespace)

	// debug API
	portDebug, err := parsePort(o.Config.DebugAPIAddr)
	if err != nil {
		return fmt.Errorf("parsing Debug port from config: %s", err)
	}

	// debug service
	debugSvc := fmt.Sprintf("%s-debug", o.Name)
	if err := c.Service.Set(ctx, debugSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Ports: []service.Port{
			{
				Name:       "debug",
				Protocol:   "TCP",
				Port:       portDebug,
				TargetPort: "debug",
			},
		},
		Selector: o.Selector,
		Type:     "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", debugSvc, o.Namespace)

	// debug service's ingress
	debugIn := fmt.Sprintf("%s-debug", o.Name)
	if err := c.Ingress.Set(ctx, debugIn, o.Namespace, ingress.Options{
		Annotations: mergeMaps(o.Annotations, o.IngressDebugAnnotations),
		Labels:      o.Labels,
		Host:        o.IngressDebugHost,
		ServiceName: debugSvc,
		ServicePort: "debug",
		Path:        "/",
	}); err != nil {
		return fmt.Errorf("set ingress in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("ingress %s is set in namespace %s\n", debugIn, o.Namespace)

	// p2p service
	portP2P, err := parsePort(o.Config.P2PAddr)
	if err != nil {
		return fmt.Errorf("parsing P2P port from config: %s", err)
	}

	p2pSvc := fmt.Sprintf("%s-p2p", o.Name)
	if err := c.Service.Set(ctx, p2pSvc, o.Namespace, service.Options{
		Annotations:           o.Annotations,
		Labels:                o.Labels,
		ExternalTrafficPolicy: "Local",
		Ports: []service.Port{
			{
				Name:       "p2p",
				Protocol:   "TCP",
				Port:       portP2P,
				TargetPort: "p2p",
			},
		},
		Selector: o.Selector,
		Type:     "NodePort",
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", p2pSvc, o.Namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", o.Name)
	if err := c.Service.Set(ctx, headlessSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Ports: []service.Port{
			{
				Name:       "api",
				Protocol:   "TCP",
				Port:       portAPI,
				TargetPort: "api",
			},
			{
				Name:       "debug",
				Protocol:   "TCP",
				Port:       portDebug,
				TargetPort: "debug",
			},
			{
				Name:       "p2p",
				Protocol:   "TCP",
				Port:       portP2P,
				TargetPort: "p2p",
			},
		},
		Selector: o.Selector,
		Type:     "ClusterIP",
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", headlessSvc, o.Namespace)

	// statefulset
	sSet := o.Name
	if err := c.StatefulSet.Set(ctx, sSet, o.Namespace, statefulset.Options{
		Annotations: o.Annotations,
		InitContainers: []statefulset.InitContainer{{
			Name:    "init-libp2p",
			Image:   "busybox:1.28",
			Command: []string{"sh", "-c", initCmd},
			VolumeMounts: []statefulset.VolumeMount{
				{
					Name:      "bee-libp2p",
					MountPath: "/tmp/bee",
				},
				{
					Name:      "data",
					MountPath: "home/bee/.bee",
				},
			},
		}},
		Containers: []statefulset.Container{{
			Name:            sSet,
			Image:           o.Image,
			ImagePullPolicy: o.ImagePullPolicy,
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
				LimitCPU:      o.LimitCPU,
				LimitMemory:   o.LimitMemory,
				RequestCPU:    o.RequestCPU,
				RequestMemory: o.RequestMemory,
			},
			SecurityContext: statefulset.SecurityContext{
				AllowPrivilegeEscalation: false,
				RunAsUser:                999,
			},
			VolumeMounts: []statefulset.VolumeMount{
				{
					Name:      "config",
					MountPath: "/home/bee/.bee.yaml",
					SubPath:   ".bee.yaml",
					ReadOnly:  true,
				},
				{
					Name:      "data",
					MountPath: "home/bee/.bee",
				},
			},
		}},
		Labels:              o.Labels,
		NodeSelector:        o.NodeSelector,
		PodManagementPolicy: o.PodManagementPolicy,
		PodSecurityContext: statefulset.PodSecurityContext{
			FSGroup: 999,
		},
		Replicas:           1,
		RestartPolicy:      o.RestartPolicy,
		Selector:           o.Selector,
		ServiceAccountName: svcAccount,
		ServiceName:        headlessSvc,
		UpdateStrategy: statefulset.UpdateStrategy{
			Type: o.UpdateStrategy,
		},
		Volumes: []statefulset.Volume{
			{
				ConfigMap: &statefulset.ConfigMapVolume{
					Name:          "config",
					ConfigMapName: configCM,
					DefaultMode:   420,
				},
			},
			{EmptyDir: &statefulset.EmptyDirVolume{
				Name: "data",
			}},
			{Secret: &statefulset.SecretVolume{
				Name:        libp2pSecret,
				SecretName:  libp2pSecret,
				DefaultMode: 420,
				Items: []statefulset.Item{{
					Key:   "libp2pKeys",
					Value: "libp2p.map",
				}},
			}},
		},
	}); err != nil {
		return fmt.Errorf("set statefulset in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("statefulset %s is set in namespace %s\n", sSet, o.Namespace)

	fmt.Printf("node %s started in namespace %s\n", o.Name, o.Namespace)
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
