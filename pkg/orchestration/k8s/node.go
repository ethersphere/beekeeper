package k8s

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

// compile check whether client implements interface
var _ orchestration.Node = (*Node)(nil)

// Node represents Bee node
type Node struct {
	name         string
	clefKey      string
	clefPassword string
	client       *bee.Client
	config       *orchestration.Config
	k8s          *k8s.Client
	libP2PKey    string
	swarmKey     string
}

// NewNode returns Bee node
func NewNode(name string, opts orchestration.NodeOptions) (n *Node) {
	n = &Node{name: name}

	if opts.Client != nil {
		n.client = opts.Client
	}
	if opts.Config != nil {
		n.config = opts.Config
	}
	if len(opts.ClefKey) > 0 {
		n.clefKey = opts.ClefKey
	}
	if len(opts.ClefPassword) > 0 {
		n.clefPassword = opts.ClefPassword
	}
	if len(opts.LibP2PKey) > 0 {
		n.libP2PKey = opts.LibP2PKey
	}
	if len(opts.SwarmKey) > 0 {
		n.swarmKey = opts.SwarmKey
	}
	if opts.K8S != nil {
		n.k8s = opts.K8S
	}

	return
}

// Name returns node's name
func (n Node) Name() string {
	return n.name
}

// Client returns node's name
func (n Node) Client() *bee.Client {
	return n.client
}

// Config returns node's config
func (n Node) Config() *orchestration.Config {
	return n.config
}

// ClefKey returns node's clefKey
func (n Node) ClefKey() string {
	return n.clefKey
}

// ClefPassword returns node's clefPassword
func (n Node) ClefPassword() string {
	return n.clefPassword
}

// LibP2PKey returns node's libP2PKey
func (n Node) LibP2PKey() string {
	return n.libP2PKey
}

// SwarmKey returns node's swarmKey
func (n Node) SwarmKey() string {
	return n.swarmKey
}

// SetSwarmKey sets node's Swarm key
func (n Node) SetSwarmKey(key string) orchestration.Node {
	n.swarmKey = key
	return n
}

// SetClefKey sets node's Clef key
func (n Node) SetClefKey(key string) orchestration.Node {
	n.clefKey = key
	return n
}

// SetClefKey sets node's Clef key
func (n Node) SetClefPassword(password string) orchestration.Node {
	n.clefPassword = password
	return n
}

//

func (n Node) Create(ctx context.Context, o orchestration.CreateOptions) (err error) {
	// bee configuration
	var config bytes.Buffer
	if err := template.Must(template.New("").Parse(configTemplate)).Execute(&config, o.Config); err != nil {
		return err
	}

	configCM := o.Name
	if _, err = n.k8s.ConfigMap.Set(ctx, configCM, o.Namespace, configmap.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Data: map[string]string{
			".bee.yaml": config.String(),
		},
	}); err != nil {
		return fmt.Errorf("set configmap in namespace %s: %w", o.Namespace, err)
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

	if _, err := n.k8s.Secret.Set(ctx, keysSecret, o.Namespace, secret.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		StringData:  keysSecretData,
	}); err != nil {
		return fmt.Errorf("set secret in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("secret %s is set in namespace %s\n", keysSecret, o.Namespace)

	// secret with clef key and pass
	clefSecretEnabled := len(o.ClefKey) > 0 && len(o.ClefPassword) > 0
	clefSecret := fmt.Sprintf("%s-clef", o.Name)
	if o.Config.ClefSignerEnable && clefSecretEnabled {
		clefSecretData := map[string]string{
			"key":      o.ClefKey,
			"password": o.ClefPassword,
		}
		if _, err := n.k8s.Secret.Set(ctx, clefSecret, o.Namespace, secret.Options{
			Annotations: o.Annotations,
			Labels:      o.Labels,
			StringData:  clefSecretData,
		}); err != nil {
			return fmt.Errorf("set secret in namespace %s: %w", o.Namespace, err)
		}
		fmt.Printf("secret %s is set in namespace %s\n", clefSecret, o.Namespace)
	}

	// service account
	svcAccount := o.Name
	if _, err := n.k8s.ServiceAccount.Set(ctx, svcAccount, o.Namespace, serviceaccount.Options{
		Annotations:      o.Annotations,
		Labels:           o.Labels,
		ImagePullSecrets: o.ImagePullSecrets,
	}); err != nil {
		return fmt.Errorf("set serviceaccount in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("serviceaccount %s is set in namespace %s\n", svcAccount, o.Namespace)

	// api service
	portAPI, err := parsePort(o.Config.APIAddr)
	if err != nil {
		return fmt.Errorf("parsing API port from config: %s", err)
	}

	apiSvc := fmt.Sprintf("%s-api", o.Name)
	if _, err := n.k8s.Service.Set(ctx, apiSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			Ports: service.Ports{
				{
					AppProtocol: "TCP",
					Name:        "api",
					Protocol:    "TCP",
					Port:        portAPI,
					TargetPort:  "api",
				},
			},
			Selector: o.Selector,
			Type:     "ClusterIP",
		},
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", apiSvc, o.Namespace)

	// api service's ingress
	apiIn := fmt.Sprintf("%s-api", o.Name)
	if _, err := n.k8s.Ingress.Set(ctx, apiIn, o.Namespace, ingress.Options{
		Annotations: mergeMaps(o.Annotations, o.IngressAnnotations),
		Labels:      o.Labels,
		Spec: ingress.Spec{
			Class: o.IngressClass,
			Rules: ingress.Rules{{
				Host: o.IngressHost,
				Paths: ingress.Paths{{
					Backend: ingress.Backend{
						ServiceName: apiSvc,
						ServicePort: "api",
					},
					Path:     "/",
					PathType: "ImplementationSpecific",
				}},
			}},
		},
	}); err != nil {
		return fmt.Errorf("set ingress in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("ingress %s is set in namespace %s\n", apiIn, o.Namespace)

	// debug API
	portDebug, err := parsePort(o.Config.DebugAPIAddr)
	if err != nil {
		return fmt.Errorf("parsing Debug port from config: %s", err)
	}

	// debug service
	debugSvc := fmt.Sprintf("%s-debug", o.Name)
	if _, err := n.k8s.Service.Set(ctx, debugSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			Ports: service.Ports{{
				AppProtocol: "TCP",
				Name:        "debug",
				Protocol:    "TCP",
				Port:        portDebug,
				TargetPort:  "debug",
			}},
			Selector: o.Selector,
			Type:     "ClusterIP",
		},
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", debugSvc, o.Namespace)

	// debug service's ingress
	debugIn := fmt.Sprintf("%s-debug", o.Name)
	if _, err := n.k8s.Ingress.Set(ctx, debugIn, o.Namespace, ingress.Options{
		Annotations: mergeMaps(o.Annotations, o.IngressDebugAnnotations),
		Labels:      o.Labels,
		Spec: ingress.Spec{
			Class: o.IngressDebugClass,
			Rules: ingress.Rules{{
				Host: o.IngressDebugHost,
				Paths: ingress.Paths{{
					Backend: ingress.Backend{
						ServiceName: debugSvc,
						ServicePort: "debug",
					},
					Path:     "/",
					PathType: "ImplementationSpecific",
				}},
			}},
		},
	}); err != nil {
		return fmt.Errorf("set ingress in namespace %s: %w", o.Namespace, err)
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
	if _, err := n.k8s.Service.Set(ctx, p2pSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			ExternalTrafficPolicy: "Local",
			Ports: setBeeNodePort(setBeeNodePortOptions{
				AppProtocol: "TCP",
				Name:        "p2p",
				Protocol:    "TCP",
				TargetPort:  "p2p",
				Port:        portP2P,
				NodePort:    nodePortP2P,
			}),
			Selector: o.Selector,
			Type:     "NodePort",
		},
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", p2pSvc, o.Namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", o.Name)
	if _, err := n.k8s.Service.Set(ctx, headlessSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			Ports: service.Ports{
				{
					AppProtocol: "TCP",
					Name:        "api",
					Protocol:    "TCP",
					Port:        portAPI,
					TargetPort:  "api",
				},
				{
					AppProtocol: "TCP",
					Name:        "debug",
					Protocol:    "TCP",
					Port:        portDebug,
					TargetPort:  "debug",
				},
				{
					AppProtocol: "TCP",
					Name:        "p2p",
					Protocol:    "TCP",
					Port:        portP2P,
					TargetPort:  "p2p",
				},
			},
			Selector: o.Selector,
			Type:     "ClusterIP",
		},
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", headlessSvc, o.Namespace)

	// statefulset
	sSet := o.Name
	clefEnabled := o.Config.ClefSignerEnable
	libP2PEnabled := len(o.LibP2PKey) > 0
	swarmEnabled := len(o.SwarmKey) > 0

	if err := n.k8s.StatefulSet.Set(ctx, sSet, o.Namespace, statefulset.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		Spec: statefulset.StatefulSetSpec{
			PodManagementPolicy: o.PodManagementPolicy,
			Replicas:            0,
			Selector:            o.Selector,
			ServiceName:         headlessSvc,
			Template: pod.PodTemplateSpec{
				Name:        sSet,
				Namespace:   o.Namespace,
				Annotations: o.Annotations,
				Labels:      o.Labels,
				Spec: pod.PodSpec{
					InitContainers: setInitContainers(setInitContainersOptions{
						ClefEnabled:         clefEnabled,
						ClefSecretEnabled:   clefSecretEnabled,
						ClefImage:           o.ClefImage,
						ClefImagePullPolicy: o.ClefImagePullPolicy,
						ClefPassword:        o.ClefPassword,
						LibP2PEnabled:       libP2PEnabled,
						SwarmEnabled:        swarmEnabled,
					}),
					Containers: setContainers(setContainersOptions{
						Name:                   sSet,
						Image:                  o.Image,
						ImagePullPolicy:        o.ImagePullPolicy,
						PortAPI:                portAPI,
						PortDebug:              portDebug,
						PortP2P:                portP2P,
						PersistenceEnabled:     o.PersistenceEnabled,
						ResourcesLimitCPU:      o.ResourcesLimitCPU,
						ResourcesLimitMemory:   o.ResourcesLimitMemory,
						ResourcesRequestCPU:    o.ResourcesRequestCPU,
						ResourcesRequestMemory: o.ResourcesRequestMemory,
						ClefEnabled:            clefEnabled,
						ClefSecretEnabled:      clefSecretEnabled,
						ClefImage:              o.ClefImage,
						ClefImagePullPolicy:    o.ClefImagePullPolicy,
						ClefPassword:           o.ClefPassword,
						LibP2PEnabled:          libP2PEnabled,
						SwarmEnabled:           swarmEnabled,
					}),
					NodeSelector: o.NodeSelector,
					PodSecurityContext: pod.PodSecurityContext{
						FSGroup: 999,
					},
					RestartPolicy:      o.RestartPolicy,
					ServiceAccountName: svcAccount,
					Volumes: setVolumes(setVolumesOptions{
						ConfigCM:           configCM,
						KeysSecret:         keysSecret,
						PersistenceEnabled: o.PersistenceEnabled,
						ClefEnabled:        clefEnabled,
						ClefSecretEnabled:  clefSecretEnabled,
						ClefSecret:         clefSecret,
						LibP2PEnabled:      libP2PEnabled,
						SwarmEnabled:       swarmEnabled,
					}),
				},
			},
			UpdateStrategy: statefulset.UpdateStrategy{
				Type: o.UpdateStrategy,
			},
			VolumeClaimTemplates: setPersistentVolumeClaims(setPersistentVolumeClaimsOptions{
				Enabled:        o.PersistenceEnabled,
				StorageClass:   o.PersistenceStorageClass,
				StorageRequest: o.PersistenceStorageRequest,
			}),
		},
	}); err != nil {
		return fmt.Errorf("set statefulset in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("statefulset %s is set in namespace %s\n", sSet, o.Namespace)

	fmt.Printf("node %s started in namespace %s\n", o.Name, o.Namespace)
	return
}

func (n Node) Delete(ctx context.Context, namespace string) (err error) {
	// statefulset
	if err := n.k8s.StatefulSet.Delete(ctx, n.name, namespace); err != nil {
		return fmt.Errorf("deleting statefulset in namespace %s: %w", namespace, err)
	}
	fmt.Printf("statefulset %s is deleted in namespace %s\n", n.name, namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", n.name)
	if err := n.k8s.Service.Delete(ctx, headlessSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", headlessSvc, namespace)

	// p2p service
	p2pSvc := fmt.Sprintf("%s-p2p", n.name)
	if err := n.k8s.Service.Delete(ctx, p2pSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", p2pSvc, namespace)

	// debug service's ingress
	debugIn := fmt.Sprintf("%s-debug", n.name)
	if err := n.k8s.Ingress.Delete(ctx, debugIn, namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", namespace, err)
	}
	fmt.Printf("ingress %s is deleted in namespace %s\n", debugIn, namespace)

	// debug service
	debugSvc := fmt.Sprintf("%s-debug", n.name)
	if err := n.k8s.Service.Delete(ctx, debugSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", debugSvc, namespace)

	// api service's ingress
	apiIn := fmt.Sprintf("%s-api", n.name)
	if err := n.k8s.Ingress.Delete(ctx, apiIn, namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", namespace, err)
	}
	fmt.Printf("ingress %s is deleted in namespace %s\n", apiIn, namespace)

	// api service
	apiSvc := fmt.Sprintf("%s-api", n.name)
	if err := n.k8s.Service.Delete(ctx, apiSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", apiSvc, namespace)

	// service account
	svcAccount := n.name
	if err := n.k8s.ServiceAccount.Delete(ctx, svcAccount, namespace); err != nil {
		return fmt.Errorf("deleting serviceaccount in namespace %s: %w", namespace, err)
	}
	fmt.Printf("serviceaccount %s is deleted in namespace %s\n", svcAccount, namespace)

	// secret with clef key
	clefSecret := fmt.Sprintf("%s-clef", n.name)
	if err := n.k8s.Secret.Delete(ctx, clefSecret, namespace); err != nil {
		return fmt.Errorf("deleting secret in namespace %s: %w", namespace, err)
	}
	fmt.Printf("secret %s is deleted in namespace %s\n", clefSecret, namespace)

	// secret with keys
	keysSecret := fmt.Sprintf("%s-keys", n.name)
	if err = n.k8s.Secret.Delete(ctx, keysSecret, namespace); err != nil {
		return fmt.Errorf("deleting secret %s in namespace %s: %w", keysSecret, namespace, err)
	}
	fmt.Printf("secret %s is deleted in namespace %s\n", keysSecret, namespace)

	// bee configuration
	configCM := n.name
	if err = n.k8s.ConfigMap.Delete(ctx, configCM, namespace); err != nil {
		return fmt.Errorf("deleting configmap %s in namespace %s: %w", configCM, namespace, err)
	}
	fmt.Printf("configmap %s is deleted in namespace %s\n", configCM, namespace)

	fmt.Printf("node %s is deleted in namespace %s\n", n.name, namespace)
	return
}

func (n Node) Ready(ctx context.Context, namespace string) (ready bool, err error) {
	r, err := n.k8s.StatefulSet.ReadyReplicas(ctx, n.name, namespace)
	if err != nil {
		return false, fmt.Errorf("statefulset %s in namespace %s ready replicas: %w", n.name, namespace, err)
	}

	return r == 1, nil
}

func (n Node) Start(ctx context.Context, namespace string) (err error) {
	err = n.k8s.StatefulSet.Scale(ctx, n.name, namespace, 1)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", n.name, namespace, err)
	}

	fmt.Printf("node %s is started in namespace %s\n", n.name, namespace)
	return
}

func (n Node) Stop(ctx context.Context, namespace string) (err error) {
	err = n.k8s.StatefulSet.Scale(ctx, n.name, namespace, 0)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", n.name, namespace, err)
	}

	fmt.Printf("node %s is stopped in namespace %s\n", n.name, namespace)
	return
}
