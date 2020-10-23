package bee

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

const (
	portHTTP = 80
)

// NodeStartOptions ...
type NodeStartOptions struct {
	// Bee configuration
	Config Config
	// Kubernetes configuration
	Name                      string
	Namespace                 string
	Annotations               map[string]string
	ClefImage                 string
	ClefImagePullPolicy       string
	ClefKey                   string
	ClefPassword              string
	Labels                    map[string]string
	LimitCPU                  string
	LimitMemory               string
	Image                     string
	ImagePullPolicy           string
	IngressAnnotations        map[string]string
	IngressHost               string
	IngressDebugAnnotations   map[string]string
	IngressDebugHost          string
	LibP2PKey                 string
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistanceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	RequestCPU                string
	RequestMemory             string
	Selector                  map[string]string
	SwarmKey                  string
	UpdateStrategy            string
}

// NodeStart ...
func (c Client) NodeStart(ctx context.Context, o NodeStartOptions) (err error) {
	// bee configuration
	var config bytes.Buffer
	if err := template.Must(template.New("").Parse(configTemplate)).Execute(&config, o.Config); err != nil {
		return err
	}

	configCM := o.Name
	if err = c.k8s.ConfigMap.Set(ctx, configCM, o.Namespace, configmap.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Data: map[string]string{
			".bee.yaml": config.String(),
		},
	}); err != nil {
		return fmt.Errorf("set configmap in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("configmap %s is set in namespace %s\n", configCM, o.Namespace)

	// secret with keys
	keysSecret := fmt.Sprintf("%s-keys", o.Name)
	keysSecretData := map[string]string{}
	if len(o.LibP2PKey) > 0 {
		keysSecretData["libp2p"] = o.LibP2PKey
	}
	if len(o.SwarmKey) > 0 {
		keysSecretData["swarm"] = o.SwarmKey
	}

	if err := c.k8s.Secret.Set(ctx, keysSecret, o.Namespace, secret.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		StringData:  keysSecretData,
	}); err != nil {
		return fmt.Errorf("set secret in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("secret %s is set in namespace %s\n", keysSecret, o.Namespace)

	// secret with clef key
	clefKeySecret := "clef-key"
	if len(o.ClefKey) > 0 {
		clefKeySecretData := map[string]string{
			"clef": o.ClefKey,
		}

		if err := c.k8s.Secret.Set(ctx, clefKeySecret, o.Namespace, secret.Options{
			Annotations: o.Annotations,
			Labels:      o.Labels,
			StringData:  clefKeySecretData,
		}); err != nil {
			return fmt.Errorf("set secret in namespace %s: %s", o.Namespace, err)
		}
		fmt.Printf("secret %s is set in namespace %s\n", clefKeySecret, o.Namespace)
	}

	// service account
	svcAccount := o.Name
	if err := c.k8s.ServiceAccount.Set(ctx, svcAccount, o.Namespace, serviceaccount.Options{
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
	if err := c.k8s.Service.Set(ctx, apiSvc, o.Namespace, service.Options{
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
	if err := c.k8s.Ingress.Set(ctx, apiIn, o.Namespace, ingress.Options{
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
	if err := c.k8s.Service.Set(ctx, debugSvc, o.Namespace, service.Options{
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
	if err := c.k8s.Ingress.Set(ctx, debugIn, o.Namespace, ingress.Options{
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

	var nodePortP2P int32
	if len(o.Config.NATAddr) > 0 {
		nodePortP2P, err = parsePort(o.Config.NATAddr)
		if err != nil {
			return fmt.Errorf("parsing NAT address from config: %s", err)
		}
	}

	p2pSvc := fmt.Sprintf("%s-p2p", o.Name)
	if err := c.k8s.Service.Set(ctx, p2pSvc, o.Namespace, service.Options{
		Annotations:           o.Annotations,
		Labels:                o.Labels,
		ExternalTrafficPolicy: "Local",
		Ports: setBeeNodePort(setBeeNodePortOptions{
			Name:       "p2p",
			Protocol:   "TCP",
			TargetPort: "p2p",
			Port:       portP2P,
			NodePort:   nodePortP2P,
		}),
		Selector: o.Selector,
		Type:     "NodePort",
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", p2pSvc, o.Namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", o.Name)
	if err := c.k8s.Service.Set(ctx, headlessSvc, o.Namespace, service.Options{
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
	clefEnabled := len(o.ClefKey) > 0
	libP2PEnabled := len(o.LibP2PKey) > 0
	swarmEnabled := len(o.SwarmKey) > 0

	if err := c.k8s.StatefulSet.Set(ctx, sSet, o.Namespace, statefulset.Options{
		Annotations: o.Annotations,
		InitContainers: setInitContainers(setInitContainersOptions{
			ClefEnabled:         clefEnabled,
			ClefImage:           o.ClefImage,
			ClefImagePullPolicy: o.ClefImagePullPolicy,
			ClefPassword:        o.ClefPassword,
			LibP2PEnabled:       libP2PEnabled,
			SwarmEnabled:        swarmEnabled,
		}),
		Containers: setContainers(setContainersOptions{
			Name:                sSet,
			Image:               o.Image,
			ImagePullPolicy:     o.ImagePullPolicy,
			LimitCPU:            o.LimitCPU,
			LimitMemory:         o.LimitMemory,
			RequestCPU:          o.RequestCPU,
			RequestMemory:       o.RequestMemory,
			PortAPI:             portAPI,
			PortDebug:           portDebug,
			PortP2P:             portP2P,
			PersistenceEnabled:  o.PersistenceEnabled,
			ClefEnabled:         clefEnabled,
			ClefImage:           o.ClefImage,
			ClefImagePullPolicy: o.ClefImagePullPolicy,
			ClefPassword:        o.ClefPassword,
			LibP2PEnabled:       libP2PEnabled,
			SwarmEnabled:        swarmEnabled,
		}),
		Labels:       o.Labels,
		NodeSelector: o.NodeSelector,
		PersistentVolumeClaims: setPersistentVolumeClaims(setPersistentVolumeClaimsOptions{
			Enabled:        o.PersistenceEnabled,
			StorageClass:   o.PersistenceStorageClass,
			StorageRequest: o.PersistanceStorageRequest,
		}),
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
		Volumes: setVolumes(setVolumesOptions{
			ConfigCM:           configCM,
			KeysSecret:         keysSecret,
			PersistenceEnabled: o.PersistenceEnabled,
			ClefEnabled:        clefEnabled,
			ClefKeySecret:      clefKeySecret,
			LibP2PEnabled:      libP2PEnabled,
			SwarmEnabled:       swarmEnabled,
		}),
	}); err != nil {
		return fmt.Errorf("set statefulset in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("statefulset %s is set in namespace %s\n", sSet, o.Namespace)

	fmt.Printf("node %s started in namespace %s\n", o.Name, o.Namespace)
	return
}
