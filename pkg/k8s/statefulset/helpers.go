package statefulset

import v1 "k8s.io/api/core/v1"

func initContainersToK8S(containers []InitContainer) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}

func containersToK8S(containers []Container) (cs []v1.Container) {
	for _, container := range containers {
		cs = append(cs, container.toK8S())
	}
	return
}

func persistentVolumeClaimsToK8S(persistentVolumeClaims []PersistentVolumeClaim, namespace string, annotations, labels map[string]string) (pvcs []v1.PersistentVolumeClaim) {
	for _, pvc := range persistentVolumeClaims {
		pvcs = append(pvcs, pvc.toK8S(namespace, annotations, labels))
	}
	return
}

func portsToK8S(ports []Port) (ps []v1.ContainerPort) {
	for _, port := range ports {
		ps = append(ps, port.toK8S())
	}
	return
}

func volumeMountsToK8S(volumeMounts []VolumeMount) (vms []v1.VolumeMount) {
	for _, volumeMount := range volumeMounts {
		vms = append(vms, volumeMount.toK8S())
	}
	return
}

func volumesToK8S(volumes []Volume) (vs []v1.Volume) {
	for _, volume := range volumes {
		vs = append(vs, volume.toK8S())
	}
	return
}
