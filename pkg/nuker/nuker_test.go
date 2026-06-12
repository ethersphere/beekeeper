package nuker

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func httpReadinessProbe(path, port string) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromString(port),
			},
		},
	}
}

func statefulSetWithContainers(containers ...corev1.Container) *v1.StatefulSet {
	return &v1.StatefulSet{
		Spec: v1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func TestRestoreReadinessProbes(t *testing.T) {
	t.Run("restores each container's own probe by name", func(t *testing.T) {
		original := statefulSetWithContainers(
			corev1.Container{Name: "bee", ReadinessProbe: httpReadinessProbe("/readiness", "api")},
			corev1.Container{Name: "swarm-proxy", ReadinessProbe: httpReadinessProbe("/readiness", "swarm-proxy")},
		)
		// target mimics the post-nuke state: probes stripped, and containers in a
		// different order to confirm the restore is name-based, not index-based.
		target := statefulSetWithContainers(
			corev1.Container{Name: "swarm-proxy"},
			corev1.Container{Name: "bee"},
		)

		restoreReadinessProbes(target, original)

		got := make(map[string]*corev1.Probe)
		for _, c := range target.Spec.Template.Spec.Containers {
			got[c.Name] = c.ReadinessProbe
		}

		if got["bee"] == nil || got["bee"].HTTPGet == nil || got["bee"].HTTPGet.Port.StrVal != "api" {
			t.Errorf("bee probe = %+v, want HTTPGet on port %q", got["bee"], "api")
		}
		if got["swarm-proxy"] == nil || got["swarm-proxy"].HTTPGet == nil || got["swarm-proxy"].HTTPGet.Port.StrVal != "swarm-proxy" {
			t.Errorf("swarm-proxy probe = %+v, want HTTPGet on port %q", got["swarm-proxy"], "swarm-proxy")
		}
	})

	t.Run("restored probes are independent copies", func(t *testing.T) {
		original := statefulSetWithContainers(
			corev1.Container{Name: "bee", ReadinessProbe: httpReadinessProbe("/readiness", "api")},
		)
		target := statefulSetWithContainers(
			corev1.Container{Name: "bee"},
		)

		restoreReadinessProbes(target, original)

		// Mutating the original's probe after the restore must not affect the
		// target. Pointer aliasing would leak the change; a deep copy isolates it.
		original.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path = "/changed"

		gotPath := target.Spec.Template.Spec.Containers[0].ReadinessProbe.HTTPGet.Path
		if gotPath != "/readiness" {
			t.Errorf("target probe path = %q, want %q (restored probe must be an independent copy)", gotPath, "/readiness")
		}
	})
}
