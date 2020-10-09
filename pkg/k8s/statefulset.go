package k8s

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
	fsGroup                  int64 = 999
	allowPrivilegeEscalation bool  = false
	runAsUser                int64 = 0
	statefulsetSpec                = appsv1.StatefulSetSpec{
		// Replicas: *int32, // defaults to 1
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/instance":   "bee",
				"app.kubernetes.io/name":       "bee",
				"app.kubernetes.io/managed-by": "beekeeper",
			},
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: annotations,
				Labels:      labels,
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
					Name:            name,
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
				RestartPolicy:      v1.RestartPolicyAlways, // v1.RestartPolicyOnFailure, v1.RestartPolicyNever
				NodeSelector:       map[string]string{"node-group": "bee-staging"},
				ServiceAccountName: name,
				SecurityContext:    &v1.PodSecurityContext{FSGroup: &fsGroup},
				ImagePullSecrets:   []v1.LocalObjectReference{},
				Affinity:           &v1.Affinity{},
				Tolerations:        []v1.Toleration{},
				Volumes: []v1.Volume{
					{Name: fmt.Sprintf("%s-libp2p", name), VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{SecretName: fmt.Sprintf("%s-libp2p", name), DefaultMode: &defaultMode, Items: []v1.KeyToPath{{Key: "libp2pKeys", Path: "libp2p.map"}}}}},
					{Name: "config", VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: name}, DefaultMode: &defaultMode}}},
					{Name: "config-file", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
					{Name: "data", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}}, // TODO: enable persistence
				},
			},
		},
		VolumeClaimTemplates: []v1.PersistentVolumeClaim{},
		ServiceName:          fmt.Sprintf("%s-headless", name),
		PodManagementPolicy:  appsv1.OrderedReadyPodManagement, // appsv1.ParallelPodManagement
		UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.OnDeleteStatefulSetStrategyType, // appsv1.RollingUpdateStatefulSetStrategyType
			// RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{Partition: *int32},
		},
	}
)

// setStatefulSet creates StatefulSet, if StatefulSet already exists updates in place
func setStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, ssSpec appsv1.StatefulSetSpec) (err error) {
	spec := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: ssSpec,
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Create(ctx, spec, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			fmt.Printf("statefulset %s already exists in the namespace %s, updating the statefulset\n", name, namespace)
			_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, spec, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}

	return
}
