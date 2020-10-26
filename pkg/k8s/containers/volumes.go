package containers

import v1 "k8s.io/api/core/v1"

// VolumeDevice ...
type VolumeDevice struct {
	Name       string
	DevicePath string
}

func (vd VolumeDevice) toK8S() v1.VolumeDevice {
	return v1.VolumeDevice{
		Name:       vd.Name,
		DevicePath: vd.DevicePath,
	}
}

func volumeDevicesToK8S(volumeDevices []VolumeDevice) (l []v1.VolumeDevice) {
	for _, volumeDevice := range volumeDevices {
		l = append(l, volumeDevice.toK8S())
	}
	return
}

// VolumeMount ...
type VolumeMount struct {
	Name      string
	MountPath string
	SubPath   string
	ReadOnly  bool
}

func (v VolumeMount) toK8S() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      v.Name,
		MountPath: v.MountPath,
		SubPath:   v.SubPath,
		ReadOnly:  v.ReadOnly,
	}
}

func volumeMountsToK8S(volumeMounts []VolumeMount) (vms []v1.VolumeMount) {
	for _, volumeMount := range volumeMounts {
		vms = append(vms, volumeMount.toK8S())
	}
	return
}
