package statefulset

import v1 "k8s.io/api/core/v1"

func initContainersToK8S(containers []InitContainer) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}
