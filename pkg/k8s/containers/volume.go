package containers

import v1 "k8s.io/api/core/v1"

// VolumeDevices represents Kubernetes VolumeDevices
type VolumeDevices []VolumeDevice

// toK8S converts VolumeDevices to Kubernetes client objects
func (vds VolumeDevices) toK8S() (l []v1.VolumeDevice) {
	if len(vds) > 0 {
		l = make([]v1.VolumeDevice, 0, len(vds))
		for _, vd := range vds {
			l = append(l, vd.toK8S())
		}
	}
	return
}

// VolumeDevice represents Kubernetes VolumeDevice
type VolumeDevice struct {
	Name       string
	DevicePath string
}

// toK8S converts VolumeDevice to Kubernetes client object
func (vd *VolumeDevice) toK8S() v1.VolumeDevice {
	return v1.VolumeDevice{
		Name:       vd.Name,
		DevicePath: vd.DevicePath,
	}
}

// VolumeMounts represents Kubernetes VolumeMounts
type VolumeMounts []VolumeMount

// toK8S converts VolumeMounts to Kubernetes client objects
func (vms VolumeMounts) toK8S() (l []v1.VolumeMount) {
	if len(vms) > 0 {
		l = make([]v1.VolumeMount, 0, len(vms))
		for _, vm := range vms {
			l = append(l, vm.toK8S())
		}
	}
	return
}

// VolumeMount represents Kubernetes VolumeMount
type VolumeMount struct {
	Name      string
	MountPath string
	SubPath   string
	ReadOnly  bool
}

// toK8S converts VolumeMount to Kubernetes client object
func (v *VolumeMount) toK8S() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      v.Name,
		MountPath: v.MountPath,
		SubPath:   v.SubPath,
		ReadOnly:  v.ReadOnly,
	}
}
