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
	clefKey   = `{"address":"fd50ede4954655b993ed69238c55219da7e81acf","crypto":{"cipher":"aes-128-ctr","ciphertext":"1c0f603b0dffe53294c7ca02c1a2800d81d855970db0df1a84cc11bc1d6cf364","cipherparams":{"iv":"11c9ac512348d7ccfe5ee59d9c9388d3"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"f6d7a0947da105fa5ef70fa298f65409d12967108c0e6260f847dc2b10455b89"},"mac":"fc6585e300ad3cb21c5f648b16b8a59ca33bcf13c58197176ffee4786628eaeb"},"id":"4911f965-b425-4011-895d-a2008f859859","version":3}`
	libp2pKey = `{"address":"aa6675fb77f3f84304a00d5ea09902d8a500364091a457cf21e05a41875d48f7","crypto":{"cipher":"aes-128-ctr","ciphertext":"93effebd3f015f496367e14218cb26d22de8f899e1d7b7686deb6ab43c876ea5","cipherparams":{"iv":"627434462c2f960d37338022d27fc92e"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"a59e72e725fe3de25dd9c55aa55a93ed0e9090b408065a7204e2f505653acb70"},"mac":"dfb1e7ad93252928a7ff21ea5b65e8a4b9bda2c2e09cb6a8ac337da7a3568b8c"},"version":3}`
	swarmKey  = `{"address":"f176839c150e52fe30e5c2b5c648465c6fdfa532","crypto":{"cipher":"aes-128-ctr","ciphertext":"352af096f0fca9dfbd20a6861bde43d988efe7f179e0a9ffd812a285fdcd63b9","cipherparams":{"iv":"613003f1f1bf93430c92629da33f8828"},"kdf":"scrypt","kdfparams":{"n":32768,"r":8,"p":1,"dklen":32,"salt":"ad1d99a4c64c95c26131e079e8c8a82221d58bf66a7ceb767c33a4c376c564b8"},"mac":"cafda1bc8ca0ffc2b22eb69afd1cf5072fd09412243443be1b0c6832f57924b6"},"version":3}`
	portHTTP  = 80
)

// NodeStartOptions ...
type NodeStartOptions struct {
	// Bee configuration
	Config Config
	// Kubernetes configuration
	Name                      string
	Namespace                 string
	Annotations               map[string]string
	ClefEnabled               bool
	Labels                    map[string]string
	LimitCPU                  string
	LimitMemory               string
	Image                     string
	ImagePullPolicy           string
	IngressAnnotations        map[string]string
	IngressHost               string
	IngressDebugAnnotations   map[string]string
	IngressDebugHost          string
	LibP2PEnabled             bool
	NodeSelector              map[string]string
	PersistenceEnabled        bool
	PersistenceStorageClass   string
	PersistanceStorageRequest string
	PodManagementPolicy       string
	RestartPolicy             string
	RequestCPU                string
	RequestMemory             string
	Selector                  map[string]string
	SwarmEnabled              bool
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
	if o.ClefEnabled {
		keysSecretData["clef"] = clefKey
	}
	if o.LibP2PEnabled {
		keysSecretData["libp2p"] = libp2pKey
	}
	if o.SwarmEnabled {
		keysSecretData["swarm"] = swarmKey
	}

	if err := c.k8s.Secret.Set(ctx, keysSecret, o.Namespace, secret.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		StringData:  keysSecretData,
	}); err != nil {
		return fmt.Errorf("set secret in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("secret %s is set in namespace %s\n", keysSecret, o.Namespace)

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
		Ports:                 setNodePort("p2p", "TCP", "p2p", portP2P, nodePortP2P),
		Selector:              o.Selector,
		Type:                  "NodePort",
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
	if err := c.k8s.StatefulSet.Set(ctx, sSet, o.Namespace, statefulset.Options{
		Annotations:            o.Annotations,
		InitContainers:         setInitContainers(o.ClefEnabled, o.LibP2PEnabled, o.SwarmEnabled),
		Containers:             setContainers(sSet, o.Image, o.ImagePullPolicy, o.LimitCPU, o.LimitMemory, o.RequestCPU, o.RequestMemory, portAPI, portDebug, portP2P, o.PersistenceEnabled, o.ClefEnabled, o.LibP2PEnabled, o.SwarmEnabled),
		Labels:                 o.Labels,
		NodeSelector:           o.NodeSelector,
		PersistentVolumeClaims: setPersistentVolumeClaims(o.PersistenceEnabled, o.PersistenceStorageClass, o.PersistanceStorageRequest),
		PodManagementPolicy:    o.PodManagementPolicy,
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
		Volumes: setVolumes(configCM, keysSecret, o.PersistenceEnabled, o.ClefEnabled, o.LibP2PEnabled, o.SwarmEnabled),
	}); err != nil {
		return fmt.Errorf("set statefulset in namespace %s: %s", o.Namespace, err)
	}
	fmt.Printf("statefulset %s is set in namespace %s\n", sSet, o.Namespace)

	fmt.Printf("node %s started in namespace %s\n", o.Name, o.Namespace)
	return
}
