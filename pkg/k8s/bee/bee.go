package bee

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

// compile check whether client implements interface
var _ k8s.Bee = (*Client)(nil)

// Client manages communication with the Kubernetes
type Client struct {
	k8s *k8s.Client
}

// ClientOptions holds optional parameters for the Client.
type ClientOptions struct {
	KubeconfigPath string
}

// NewClient returns Kubernetes clientset
func NewClient(k8s *k8s.Client) (c *Client) {
	return &Client{
		k8s: k8s,
	}
}

// Create creates Bee node in the cluster
func (c *Client) Create(ctx context.Context, o k8s.CreateOptions) (err error) {
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

	if err := c.k8s.Secret.Set(ctx, keysSecret, o.Namespace, secret.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		StringData:  keysSecretData,
	}); err != nil {
		return fmt.Errorf("set secret in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("secret %s is set in namespace %s\n", keysSecret, o.Namespace)

	// secret with clef key and pass
	clefSecret := fmt.Sprintf("%s-clef", o.Name)
	if len(o.ClefKey) > 0 && len(o.ClefPassword) > 0 {
		clefSecretData := map[string]string{
			"key":      o.ClefKey,
			"password": o.ClefPassword,
		}
		if err := c.k8s.Secret.Set(ctx, clefSecret, o.Namespace, secret.Options{
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
	if err := c.k8s.ServiceAccount.Set(ctx, svcAccount, o.Namespace, serviceaccount.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
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
	if err := c.k8s.Service.Set(ctx, apiSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			Ports: service.Ports{
				{
					Name:       "api",
					Protocol:   "TCP",
					Port:       portHTTP,
					TargetPort: "api",
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
	if err := c.k8s.Ingress.Set(ctx, apiIn, o.Namespace, ingress.Options{
		Annotations: mergeMaps(o.Annotations, o.IngressAnnotations),
		Labels:      o.Labels,
		Spec: ingress.Spec{
			Backend: ingress.Backend{
				ServiceName: apiSvc,
				ServicePort: "api",
			},
			Rules: ingress.Rules{{
				Host: o.IngressHost,
				Paths: ingress.Paths{{
					Backend: ingress.Backend{
						ServiceName: apiSvc,
						ServicePort: "api",
					},
					Path: "/",
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
	if err := c.k8s.Service.Set(ctx, debugSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			Ports: service.Ports{{
				Name:       "debug",
				Protocol:   "TCP",
				Port:       portDebug,
				TargetPort: "debug",
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
	if err := c.k8s.Ingress.Set(ctx, debugIn, o.Namespace, ingress.Options{
		Annotations: mergeMaps(o.Annotations, o.IngressDebugAnnotations),
		Labels:      o.Labels,
		Spec: ingress.Spec{
			Backend: ingress.Backend{
				ServiceName: debugSvc,
				ServicePort: "debug",
			},
			Rules: ingress.Rules{{
				Host: o.IngressDebugHost,
				Paths: ingress.Paths{{
					Backend: ingress.Backend{
						ServiceName: debugSvc,
						ServicePort: "debug",
					},
					Path: "/",
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
	if err := c.k8s.Service.Set(ctx, p2pSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
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
		},
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", p2pSvc, o.Namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", o.Name)
	if err := c.k8s.Service.Set(ctx, headlessSvc, o.Namespace, service.Options{
		Annotations: o.Annotations,
		Labels:      o.Labels,
		ServiceSpec: service.Spec{
			Ports: service.Ports{
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
		},
	}); err != nil {
		return fmt.Errorf("set service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is set in namespace %s\n", headlessSvc, o.Namespace)

	// statefulset
	sSet := o.Name
	clefEnabled := len(o.ClefKey) > 0
	libP2PEnabled := len(o.LibP2PKey) > 0
	swarmEnabled := len(o.SwarmKey) > 0

	if err := c.k8s.StatefulSet.Set(ctx, sSet, o.Namespace, statefulset.Options{
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
				StorageRequest: o.PersistanceStorageRequest,
			}),
		},
	}); err != nil {
		return fmt.Errorf("set statefulset in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("statefulset %s is set in namespace %s\n", sSet, o.Namespace)

	fmt.Printf("node %s started in namespace %s\n", o.Name, o.Namespace)
	return
}

// Delete deletes Bee node from the cluster
func (c *Client) Delete(ctx context.Context, name, namespace string) (err error) {
	// statefulset
	if err := c.k8s.StatefulSet.Delete(ctx, name, namespace); err != nil {
		return fmt.Errorf("deleting statefulset in namespace %s: %w", namespace, err)
	}
	fmt.Printf("statefulset %s is deleted in namespace %s\n", name, namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", name)
	if err := c.k8s.Service.Delete(ctx, headlessSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", headlessSvc, namespace)

	// p2p service
	p2pSvc := fmt.Sprintf("%s-p2p", name)
	if err := c.k8s.Service.Delete(ctx, p2pSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", p2pSvc, namespace)

	// debug service's ingress
	debugIn := fmt.Sprintf("%s-debug", name)
	if err := c.k8s.Ingress.Delete(ctx, debugIn, namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", namespace, err)
	}
	fmt.Printf("ingress %s is deleted in namespace %s\n", debugIn, namespace)

	// debug service
	debugSvc := fmt.Sprintf("%s-debug", name)
	if err := c.k8s.Service.Delete(ctx, debugSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", debugSvc, namespace)

	// api service's ingress
	apiIn := fmt.Sprintf("%s-api", name)
	if err := c.k8s.Ingress.Delete(ctx, apiIn, namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", namespace, err)
	}
	fmt.Printf("ingress %s is deleted in namespace %s\n", apiIn, namespace)

	// api service
	apiSvc := fmt.Sprintf("%s-api", name)
	if err := c.k8s.Service.Delete(ctx, apiSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", apiSvc, namespace)

	// service account
	svcAccount := name
	if err := c.k8s.ServiceAccount.Delete(ctx, svcAccount, namespace); err != nil {
		return fmt.Errorf("deleting serviceaccount in namespace %s: %w", namespace, err)
	}
	fmt.Printf("serviceaccount %s is deleted in namespace %s\n", svcAccount, namespace)

	// secret with clef key
	clefSecret := fmt.Sprintf("%s-clef", name)
	if err := c.k8s.Secret.Delete(ctx, clefSecret, namespace); err != nil {
		return fmt.Errorf("deleting secret in namespace %s: %w", namespace, err)
	}
	fmt.Printf("secret %s is deleted in namespace %s\n", clefSecret, namespace)

	// secret with keys
	keysSecret := fmt.Sprintf("%s-keys", name)
	if err = c.k8s.Secret.Delete(ctx, keysSecret, namespace); err != nil {
		return fmt.Errorf("deleting secret %s in namespace %s: %w", keysSecret, namespace, err)
	}
	fmt.Printf("secret %s is deleted in namespace %s\n", keysSecret, namespace)

	// bee configuration
	configCM := name
	if err = c.k8s.ConfigMap.Delete(ctx, configCM, namespace); err != nil {
		return fmt.Errorf("deleting configmap %s in namespace %s: %w", configCM, namespace, err)
	}
	fmt.Printf("configmap %s is deleted in namespace %s\n", configCM, namespace)

	fmt.Printf("node %s is deleted in namespace %s\n", name, namespace)
	return
}

// Ready gets Bee node's readiness
func (c *Client) Ready(ctx context.Context, name, namespace string) (ready bool, err error) {
	r, err := c.k8s.StatefulSet.ReadyReplicas(ctx, name, namespace)
	if err != nil {
		return false, fmt.Errorf("statefulset %s in namespace %s ready replicas: %w", name, namespace, err)
	}

	return r == 1, nil
}

// RunningNodes returns list of running nodes
// TODO: filter by labels
func (c *Client) RunningNodes(ctx context.Context, namespace string) (running []string, err error) {
	running, err = c.k8s.StatefulSet.RunningStatefulSets(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("running statefulsets in namespace %s: %w", namespace, err)
	}
	return
}

// Start starts Bee node in the cluster
func (c *Client) Start(ctx context.Context, name, namespace string) (err error) {
	err = c.k8s.StatefulSet.Scale(ctx, name, namespace, 1)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", name, namespace, err)
	}

	fmt.Printf("node %s is started in namespace %s\n", name, namespace)
	return
}

// Stop stops Bee node in the cluster
func (c *Client) Stop(ctx context.Context, name, namespace string) (err error) {
	err = c.k8s.StatefulSet.Scale(ctx, name, namespace, 0)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", name, namespace, err)
	}

	fmt.Printf("node %s is stopped in namespace %s\n", name, namespace)
	return
}

// StoppedNodes returns list of stopped nodes
// TODO: filter by labels
func (c *Client) StoppedNodes(ctx context.Context, namespace string) (stopped []string, err error) {
	stopped, err = c.k8s.StatefulSet.StoppedStatefulSets(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("stopped statefulsets in namespace %s: %w", namespace, err)
	}
	return
}
