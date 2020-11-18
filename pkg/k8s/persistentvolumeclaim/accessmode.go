package persistentvolumeclaim

import v1 "k8s.io/api/core/v1"

// AccessModes represents Kubernetes AccessModes
type AccessModes []AccessMode

// toK8S converts AccessModes to Kuberntes client objects
func (ams AccessModes) toK8S() (l []v1.PersistentVolumeAccessMode) {
	l = make([]v1.PersistentVolumeAccessMode, 0, len(ams))

	for _, am := range ams {
		l = append(l, am.toK8S())
	}

	return
}

// AccessMode represents Kubernetes AccessMode
type AccessMode string

// toK8S converts AccessMode to Kuberntes client object
func (a *AccessMode) toK8S() v1.PersistentVolumeAccessMode {
	return v1.PersistentVolumeAccessMode(*a)
}
