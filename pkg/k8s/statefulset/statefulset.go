package statefulset

import (
	"context"
	"fmt"

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
	Annotations         map[string]string
	Labels              map[string]string
	Replicas            int32
	Selector            map[string]string
	InitContainers      []InitContainer
	Containers          []Container
	RestartPolicy       string
	ServiceAccountName  string
	ServiceName         string
	NodeSelector        map[string]string
	PodManagementPolicy string
	PodSecurityContext  PodSecurityContext
	UpdateStrategy      UpdateStrategy
	Volumes             []Volume
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
			Replicas: &o.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: o.Selector},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Annotations: o.Annotations,
					Labels:      o.Labels,
				},
				Spec: v1.PodSpec{
					InitContainers:     initContainersToK8S(o.InitContainers),
					Containers:         containersToK8S(o.Containers),
					RestartPolicy:      v1.RestartPolicy(o.RestartPolicy),
					NodeSelector:       o.NodeSelector,
					ServiceAccountName: o.ServiceAccountName,
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: &o.PodSecurityContext.FSGroup,
					},
					Volumes: volumesToK8S(o.Volumes),
				},
			},
			ServiceName:         o.ServiceName,
			PodManagementPolicy: appsv1.PodManagementPolicyType(o.PodManagementPolicy),
			UpdateStrategy:      o.UpdateStrategy.toK8S(),
		},
	}

	_, err = c.clientset.AppsV1().StatefulSets(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			fmt.Printf("statefulset %s already exists in the namespace %s, updating the statefulset\n", name, namespace)
			_, err = c.clientset.AppsV1().StatefulSets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

// PodSecurityContext ...
type PodSecurityContext struct {
	FSGroup int64
}

// UpdateStrategy ...
type UpdateStrategy struct {
	Type                   string
	RollingUpdatePartition int32
}

func (u UpdateStrategy) toK8S() appsv1.StatefulSetUpdateStrategy {
	if u.Type == "OnDelete" {
		return appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.OnDeleteStatefulSetStrategyType,
		}
	}

	return appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
			Partition: &u.RollingUpdatePartition,
		},
	}
}

func volumesToK8S(volumes []Volume) (vs []v1.Volume) {
	for _, volume := range volumes {
		vs = append(vs, volume.toK8S())
	}
	return
}
