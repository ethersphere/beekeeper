package k8s

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/configmap"
	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/secret"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/serviceaccount"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
)

var _ orchestration.NodeOrchestrator = (*nodeOrchestrator)(nil)

type nodeOrchestrator struct {
	k8s *k8s.Client
	log logging.Logger
}

// newNodeOrchestrator returns a new Kubernetes Bee node orchestrator.
func newNodeOrchestrator(k8s *k8s.Client, log logging.Logger) orchestration.NodeOrchestrator {
	return &nodeOrchestrator{
		k8s: k8s,
		log: log,
	}
}

// RunningNodes implements orchestration.NodeOrchestrator.
func (n *nodeOrchestrator) RunningNodes(ctx context.Context, namespace string) (running []string, err error) {
	running, err = n.k8s.StatefulSet.RunningStatefulSets(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("running statefulsets in namespace %s: %w", namespace, err)
	}
	return
}

// StoppedNodes implements orchestration.NodeOrchestrator.
func (n *nodeOrchestrator) StoppedNodes(ctx context.Context, namespace string) (stopped []string, err error) {
	stopped, err = n.k8s.StatefulSet.StoppedStatefulSets(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("stopped statefulsets in namespace %s: %w", namespace, err)
	}
	return
}

// Create
func (n *nodeOrchestrator) Create(ctx context.Context, o orchestration.CreateOptions) (err error) {
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
	n.log.Infof("configmap %s is set in namespace %s", configCM, o.Namespace)

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
	n.log.Infof("secret %s is set in namespace %s", keysSecret, o.Namespace)

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
		n.log.Infof("secret %s is set in namespace %s", clefSecret, o.Namespace)
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
	n.log.Infof("serviceaccount %s is set in namespace %s", svcAccount, o.Namespace)

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
	n.log.Infof("service %s is set in namespace %s", apiSvc, o.Namespace)

	if o.IngressClass == "traefik" {
		// api service's ingressroute
		apiIn := fmt.Sprintf("%s-api", o.Name)
		if _, err := n.k8s.IngressRoute.Set(ctx, apiIn, o.Namespace, ingressroute.Options{
			Annotations: mergeMaps(o.Annotations, o.IngressAnnotations),
			Labels:      o.Labels,
			Spec: ingressroute.IngressRouteSpec{
				Routes: []ingressroute.Route{
					{
						Kind:  "Rule",
						Match: fmt.Sprintf("Host(\"%s.localhost\") && PathPrefix(\"/\")", o.Name),
						Services: []ingressroute.Service{
							{
								Kind:      "Service",
								Name:      apiIn,
								Namespace: "local",
								Port:      "api",
							},
						},
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("set ingressroute in namespace %s: %w", o.Namespace, err)
		}
		n.log.Infof("ingressroute %s is set in namespace %s", apiIn, o.Namespace)
	} else {
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
							ServiceName:     apiSvc,
							ServicePortName: "api",
						},
						Path:     "/",
						PathType: "ImplementationSpecific",
					}},
				}},
			},
		}); err != nil {
			return fmt.Errorf("set ingress in namespace %s: %w", o.Namespace, err)
		}
		n.log.Infof("ingress %s is set in namespace %s", apiIn, o.Namespace)
	}

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
	n.log.Infof("service %s is set in namespace %s", p2pSvc, o.Namespace)

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
	n.log.Infof("service %s is set in namespace %s", headlessSvc, o.Namespace)

	// statefulset
	sSet := o.Name
	clefEnabled := o.Config.ClefSignerEnable
	libP2PEnabled := len(o.LibP2PKey) > 0
	swarmEnabled := len(o.SwarmKey) > 0

	if _, err := n.k8s.StatefulSet.Set(ctx, sSet, o.Namespace, statefulset.Options{
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
	n.log.Infof("statefulset %s is set in namespace %s", sSet, o.Namespace)

	return
}

func (n *nodeOrchestrator) Delete(ctx context.Context, name string, namespace string) (err error) {
	// statefulset
	if err := n.k8s.StatefulSet.Delete(ctx, name, namespace); err != nil {
		return fmt.Errorf("deleting statefulset in namespace %s: %w", namespace, err)
	}
	n.log.Infof("statefulset %s is deleted in namespace %s", name, namespace)

	// headless service
	headlessSvc := fmt.Sprintf("%s-headless", name)
	if err := n.k8s.Service.Delete(ctx, headlessSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	n.log.Infof("service %s is deleted in namespace %s", headlessSvc, namespace)

	// p2p service
	p2pSvc := fmt.Sprintf("%s-p2p", name)
	if err := n.k8s.Service.Delete(ctx, p2pSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	n.log.Infof("service %s is deleted in namespace %s", p2pSvc, namespace)

	// api service's ingress
	apiIn := fmt.Sprintf("%s-api", name)
	if err := n.k8s.Ingress.Delete(ctx, apiIn, namespace); err != nil {
		return fmt.Errorf("deleting ingress in namespace %s: %w", namespace, err)
	}
	n.log.Infof("ingress %s is deleted in namespace %s", apiIn, namespace)

	// api service's ingress route
	if err := n.k8s.IngressRoute.Delete(ctx, apiIn, namespace); err != nil {
		return fmt.Errorf("deleting ingress route in namespace %s: %w", namespace, err)
	}
	n.log.Infof("ingress route %s is deleted in namespace %s", apiIn, namespace)

	// api service
	apiSvc := fmt.Sprintf("%s-api", name)
	if err := n.k8s.Service.Delete(ctx, apiSvc, namespace); err != nil {
		return fmt.Errorf("deleting service in namespace %s: %w", namespace, err)
	}
	n.log.Infof("service %s is deleted in namespace %s", apiSvc, namespace)

	// service account
	svcAccount := name
	if err := n.k8s.ServiceAccount.Delete(ctx, svcAccount, namespace); err != nil {
		return fmt.Errorf("deleting serviceaccount in namespace %s: %w", namespace, err)
	}
	n.log.Infof("serviceaccount %s is deleted in namespace %s", svcAccount, namespace)

	// secret with clef key
	clefSecret := fmt.Sprintf("%s-clef", name)
	if err := n.k8s.Secret.Delete(ctx, clefSecret, namespace); err != nil {
		return fmt.Errorf("deleting secret in namespace %s: %w", namespace, err)
	}
	n.log.Infof("secret %s is deleted in namespace %s", clefSecret, namespace)

	// secret with keys
	keysSecret := fmt.Sprintf("%s-keys", name)
	if err = n.k8s.Secret.Delete(ctx, keysSecret, namespace); err != nil {
		return fmt.Errorf("deleting secret %s in namespace %s: %w", keysSecret, namespace, err)
	}
	n.log.Infof("secret %s is deleted in namespace %s", keysSecret, namespace)

	// bee configuration
	configCM := name
	if err = n.k8s.ConfigMap.Delete(ctx, configCM, namespace); err != nil {
		return fmt.Errorf("deleting configmap %s in namespace %s: %w", configCM, namespace, err)
	}
	n.log.Infof("configmap %s is deleted in namespace %s", configCM, namespace)

	n.log.Infof("node %s is deleted in namespace %s", name, namespace)
	return
}

func (n *nodeOrchestrator) Ready(ctx context.Context, name string, namespace string) (ready bool, err error) {
	// r, err := n.k8s.StatefulSet.ReadyReplicas(ctx, name, namespace)
	r, err := n.k8s.StatefulSet.ReadyReplicasWatch(ctx, name, namespace)
	if err != nil {
		return false, fmt.Errorf("statefulset %s in namespace %s ready replicas: %w", name, namespace, err)
	}

	return r == 1, nil
}

func (n *nodeOrchestrator) Start(ctx context.Context, name string, namespace string) (err error) {
	_, err = n.k8s.StatefulSet.Scale(ctx, name, namespace, 1)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", name, namespace, err)
	}

	n.log.Infof("node %s is started in namespace %s", name, namespace)
	return
}

func (n *nodeOrchestrator) Stop(ctx context.Context, name string, namespace string) (err error) {
	_, err = n.k8s.StatefulSet.Scale(ctx, name, namespace, 0)
	if err != nil {
		return fmt.Errorf("scale statefulset %s in namespace %s: %w", name, namespace, err)
	}

	n.log.Infof("node %s is stopped in namespace %s", name, namespace)
	return
}
