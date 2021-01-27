package bee

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
)

const (
	portHTTP = 80
)

// CreateOptions represents available options for creating node
type CreateOptions struct {
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

// Create creates Bee node in the cluster
func (c *Client) Create(ctx context.Context, o CreateOptions) (err error) {
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

	// secret with clef key
	clefKeySecret := fmt.Sprintf("%s-clef-key", o.Name)
	if len(o.ClefKey) > 0 {
		clefKeySecretData := map[string]string{
			"clef": o.ClefKey,
		}

		if err := c.k8s.Secret.Set(ctx, clefKeySecret, o.Namespace, secret.Options{
			Annotations: o.Annotations,
			Labels:      o.Labels,
			StringData:  clefKeySecretData,
		}); err != nil {
			return fmt.Errorf("set secret in namespace %s: %w", o.Namespace, err)
		}
		fmt.Printf("secret %s is set in namespace %s\n", clefKeySecret, o.Namespace)
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
						ClefKeySecret:      clefKeySecret,
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

// DeleteOptions represents available options for starting node
type DeleteOptions struct {
	Name      string
	Namespace string
}

// Delete deletes Bee node from the cluster
func (c *Client) Delete(ctx context.Context, o DeleteOptions) (err error) {
	// statefulset
	sSet := o.Name
	if err := c.k8s.StatefulSet.Delete(ctx, sSet, o.Namespace); err != nil {
		return fmt.Errorf("deleting statefulset in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("statefulset %s is deleted in namespace %s\n", sSet, o.Namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", o.Name)
	if err := c.k8s.Service.Delete(ctx, headlessSvc, o.Namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", headlessSvc, o.Namespace)

	// p2p service
	p2pSvc := fmt.Sprintf("%s-p2p", o.Name)
	if err := c.k8s.Service.Delete(ctx, p2pSvc, o.Namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", p2pSvc, o.Namespace)

	// debug service's ingress
	debugIn := fmt.Sprintf("%s-debug", o.Name)
	if err := c.k8s.Ingress.Delete(ctx, debugIn, o.Namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("ingress %s is deleted in namespace %s\n", debugIn, o.Namespace)

	// debug service
	debugSvc := fmt.Sprintf("%s-debug", o.Name)
	if err := c.k8s.Service.Delete(ctx, debugSvc, o.Namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", debugSvc, o.Namespace)

	// api service's ingress
	apiIn := fmt.Sprintf("%s-api", o.Name)
	if err := c.k8s.Ingress.Delete(ctx, apiIn, o.Namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("ingress %s is deleted in namespace %s\n", apiIn, o.Namespace)

	// api service
	apiSvc := fmt.Sprintf("%s-api", o.Name)
	if err := c.k8s.Service.Delete(ctx, apiSvc, o.Namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("service %s is deleted in namespace %s\n", apiSvc, o.Namespace)

	// service account
	svcAccount := o.Name
	if err := c.k8s.ServiceAccount.Delete(ctx, svcAccount, o.Namespace); err != nil {
		return fmt.Errorf("deleting serviceaccount in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("serviceaccount %s is deleted in namespace %s\n", svcAccount, o.Namespace)

	// secret with clef key
	clefKeySecret := fmt.Sprintf("%s-clef-key", o.Name)
	if err := c.k8s.Secret.Delete(ctx, clefKeySecret, o.Namespace); err != nil {
		return fmt.Errorf("deleting secret in namespace %s: %w", o.Namespace, err)
	}
	fmt.Printf("secret %s is deleted in namespace %s\n", clefKeySecret, o.Namespace)

	// secret with keys
	keysSecret := fmt.Sprintf("%s-keys", o.Name)
	if err = c.k8s.Secret.Delete(ctx, keysSecret, o.Namespace); err != nil {
		return fmt.Errorf("deleting secret %s in namespace %s: %w", keysSecret, o.Namespace, err)
	}
	fmt.Printf("secret %s is deleted in namespace %s\n", keysSecret, o.Namespace)

	// bee configuration
	configCM := o.Name
	if err = c.k8s.ConfigMap.Delete(ctx, configCM, o.Namespace); err != nil {
		return fmt.Errorf("deleting configmap %s in namespace %s: %w", configCM, o.Namespace, err)
	}
	fmt.Printf("configmap %s is deleted in namespace %s\n", configCM, o.Namespace)

	fmt.Printf("node %s is deleted in namespace %s\n", o.Name, o.Namespace)
	return
}

// ReadyOptions represents available options for getting node's readiness
type ReadyOptions struct {
	Name      string
	Namespace string
}

// Ready gets Bee node's readiness
func (c *Client) Ready(ctx context.Context, o ReadyOptions) (ready bool, err error) {
	r, err := c.k8s.StatefulSet.ReadyReplicas(ctx, o.Name, o.Namespace)
	if err != nil {
		return false, fmt.Errorf("statefulset %s in namespace %s ready replicas: %w", o.Name, o.Namespace, err)
	}

	return r == 1, nil
}

// StartOptions represents available options for starting node
type StartOptions struct {
	Name      string
	Namespace string
}

// Start starts Bee node in the cluster
func (c *Client) Start(ctx context.Context, o StopOptions) (err error) {
	err = c.k8s.StatefulSet.Scale(ctx, o.Name, o.Namespace, 1)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", o.Name, o.Namespace, err)
	}

	fmt.Printf("node %s is started in namespace %s\n", o.Name, o.Namespace)
	return
}

// StartedNodes returns list of started nodes
// TODO: filter by labels
func (c *Client) StartedNodes(ctx context.Context, namespace string) (started []string, err error) {
	started, err = c.k8s.StatefulSet.StartedStatefulSets(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("started statefulsets in namespace %s: %w", namespace, err)
	}
	return
}

// StopOptions represents available options for stopping node
type StopOptions struct {
	Name      string
	Namespace string
}

// Stop stops Bee node in the cluster
func (c *Client) Stop(ctx context.Context, o StopOptions) (err error) {
	err = c.k8s.StatefulSet.Scale(ctx, o.Name, o.Namespace, 0)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", o.Name, o.Namespace, err)
	}

	fmt.Printf("node %s is stopped in namespace %s\n", o.Name, o.Namespace)
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
