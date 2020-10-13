package statefulset

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	defaultMode              int32 = 0420
	allowPrivilegeEscalation bool  = false
	runAsUser                int64 = 0
)

// Options represents statefulset's options
type Options struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Replicas    int32
	Selector    map[string]string
	// InitContainers string
	// Containers     string
	RestartPolicy       string
	ServiceAccountName  string
	ServiceName         string
	NodeSelector        map[string]string
	PodManagementPolicy string
	PodSecurityContext  PodSecurityContext
	UpdateStrategy      UpdateStrategy
	Volumes             []Volume
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

// Set creates StatefulSet, if StatefulSet already exists updates in place
func Set(ctx context.Context, clientset *kubernetes.Clientset, o Options) (err error) {
	spec := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.Name,
			Namespace:   o.Namespace,
			Annotations: o.Annotations,
			Labels:      o.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &o.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: o.Selector},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        o.Name,
					Namespace:   o.Namespace,
					Annotations: o.Annotations,
					Labels:      o.Labels,
				},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{{
						Name:    "init-libp2p",
						Image:   "busybox:1.28",
						Command: []string{"sh", "-c", `export INDEX=$(echo $(hostname) | rev | cut -d'-' -f 1 | rev); mkdir -p /home/bee/.bee/keys; chown -R 999:999 /home/bee/.bee/keys; export KEY=$(cat /tmp/bee/libp2p.map | grep bee-${INDEX}: | cut -d' ' -f2); if [ -z "${KEY}" ]; then exit 0; fi; printf '%s' "${KEY}" > /home/bee/.bee/keys/libp2p.key; echo 'node initialization done';`},
						VolumeMounts: []v1.VolumeMount{
							{Name: "bee-libp2p", MountPath: "/tmp/bee"},
							{Name: "data", MountPath: "home/bee/.bee"},
						},
					}},
					Containers: []v1.Container{{
						Name:            o.Name,
						Image:           "ethersphere/bee:latest",
						ImagePullPolicy: v1.PullAlways, // v1.PullNever, v1.PullIfNotPresent
						Command:         []string{"bee", "start", "--config=.bee.yaml"},
						LivenessProbe: &v1.Probe{
							Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{
								Path: "/health",
								Port: intstr.IntOrString{Type: intstr.String, StrVal: "debug"},
							}},
						},
						Ports: []v1.ContainerPort{
							{
								Name:          "api",
								ContainerPort: 8080,
								Protocol:      "TCP",
							},
							{
								Name:          "p2p",
								ContainerPort: 7070,
								Protocol:      "TCP",
							},
							{
								Name:          "debug",
								ContainerPort: 6060,
								Protocol:      "TCP",
							},
						},
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{HTTPGet: &v1.HTTPGetAction{
								Path: "/readiness",
								Port: intstr.IntOrString{Type: intstr.String, StrVal: "debug"},
							}},
						},
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU:    resource.Quantity{Format: "1"},
								v1.ResourceMemory: resource.Quantity{Format: "2Gi"},
							},
							Requests: v1.ResourceList{
								v1.ResourceCPU:    resource.Quantity{Format: "750m"},
								v1.ResourceMemory: resource.Quantity{Format: "1Gi"},
							},
						},
						SecurityContext: &v1.SecurityContext{
							AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							RunAsUser:                &runAsUser,
						},
						VolumeMounts: []v1.VolumeMount{
							{Name: "config", MountPath: "/home/bee/.bee.yaml", SubPath: ".bee.yaml", ReadOnly: true},
							{Name: "data", MountPath: "home/bee/.bee"},
						},
					}},
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
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.StatefulSetUpdateStrategyType(o.UpdateStrategy.Type),
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
					Partition: &o.UpdateStrategy.RollingUpdatePartition,
				},
			},
		},
	}

	_, err = clientset.AppsV1().StatefulSets(o.Namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("statefulset %s already exists in the namespace %s, updating the statefulset\n", o.Name, o.Namespace)
			_, err = clientset.AppsV1().StatefulSets(o.Namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}

func volumesToK8S(volumes []Volume) (vs []v1.Volume) {
	for _, volume := range volumes {
		vs = append(vs, volume.toK8S())
	}
	return
}
