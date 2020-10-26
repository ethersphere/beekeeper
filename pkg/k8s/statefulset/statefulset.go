package statefulset

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/k8s/containers"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client manages communication with the Kubernetes StatefulSet.
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient constructs a new Client.
func NewClient(clientset *kubernetes.Clientset) *Client {
	return &Client{
		clientset: clientset,
	}
}

// Options holds optional parameters for the Client.
type Options struct {
	Annotations                   map[string]string
	Labels                        map[string]string
	AutomountServiceAccountToken  bool
	Containers                    containers.Containers
	DNSPolicy                     string
	EnableServiceLinks            bool
	Hostname                      string
	HostNetwork                   bool
	HostPID                       bool
	HostIPC                       bool
	InitContainers                containers.Containers
	NodeName                      string
	NodeSelector                  map[string]string
	PersistentVolumeClaims        PersistentVolumeClaims
	PodManagementPolicy           string
	PodSecurityContext            PodSecurityContext
	Priority                      int32
	PriorityClassName             string
	Replicas                      int32
	RestartPolicy                 string
	RevisionHistoryLimit          int32
	SchedulerName                 string
	Selector                      map[string]string
	ServiceAccountName            string
	ServiceName                   string
	ShareProcessNamespace         bool
	Subdomain                     string
	TerminationGracePeriodSeconds int64
	UpdateStrategy                UpdateStrategy
	Volumes                       Volumes
}

// Set creates StatefulSet, if StatefulSet already exists updates in place
func (c Client) Set(ctx context.Context, name, namespace string, o Options) (err error) {
	spec := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy:  appsv1.PodManagementPolicyType(o.PodManagementPolicy),
			Replicas:             &o.Replicas,
			RevisionHistoryLimit: &o.RevisionHistoryLimit,
			Selector:             &metav1.LabelSelector{MatchLabels: o.Selector},
			ServiceName:          o.ServiceName,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Annotations: o.Annotations,
					Labels:      o.Labels,
				},
				Spec: v1.PodSpec{
					// TODO: Affinity *Affinity
					AutomountServiceAccountToken: &o.AutomountServiceAccountToken,
					Containers:                   o.Containers.ToK8S(),
					// TODO: DNSConfig *PodDNSConfig
					DNSPolicy:          v1.DNSPolicy(o.DNSPolicy),
					EnableServiceLinks: &o.EnableServiceLinks,
					// TODO: EphemeralContainers: o.EphemeralContainers.ToK8S(),
					// TODO: HostAliases []HostAlias
					Hostname:    o.Hostname,
					HostNetwork: o.HostNetwork,
					HostPID:     o.HostPID,
					HostIPC:     o.HostIPC,
					// TODO: ImagePullSecrets []LocalObjectReference
					InitContainers: o.InitContainers.ToK8S(),
					NodeName:       o.NodeName,
					NodeSelector:   o.NodeSelector,
					// TODO: Overhead ResourceList
					// TODO: PreemptionPolicy *PreemptionPolicy
					Priority:          &o.Priority,
					PriorityClassName: o.PriorityClassName,
					// TODO: ReadinessGates []PodReadinessGate
					RestartPolicy:                 v1.RestartPolicy(o.RestartPolicy),
					SchedulerName:                 o.SchedulerName,
					SecurityContext:               o.PodSecurityContext.toK8S(),
					ServiceAccountName:            o.ServiceAccountName,
					ShareProcessNamespace:         &o.ShareProcessNamespace,
					Subdomain:                     o.Subdomain,
					TerminationGracePeriodSeconds: &o.TerminationGracePeriodSeconds,
					// TODO: Tolerations []Toleration
					// TODO: TopologySpreadConstraints []TopologySpreadConstraint
					Volumes: o.Volumes.toK8S(),
				},
			},
			UpdateStrategy:       o.UpdateStrategy.toK8S(),
			VolumeClaimTemplates: o.PersistentVolumeClaims.toK8S(namespace, o.Annotations, o.Labels),
		},
	}

	_, err = c.clientset.AppsV1().StatefulSets(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(111, err)
		if !errors.IsNotFound(err) {
			fmt.Printf("statefulset %s already exists in the namespace %s, updating the statefulset\n", name, namespace)
			_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			fmt.Println(222, err)
			if err != nil {
				return err
			}
		}
		fmt.Println(333, err)
		return err
	}

	return
}
