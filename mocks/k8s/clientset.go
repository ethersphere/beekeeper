package mocks

import (
	discovery "k8s.io/client-go/discovery"
	admissionregistrationv1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	internalv1alpha1 "k8s.io/client-go/kubernetes/typed/apiserverinternal/v1alpha1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authenticationv1beta1 "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	authorizationv1beta1 "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	autoscalingv1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	autoscalingv2beta1 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	batchv1beta1 "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	certificatesv1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	certificatesv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	coordinationv1beta1 "k8s.io/client-go/kubernetes/typed/coordination/v1beta1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	discoveryv1 "k8s.io/client-go/kubernetes/typed/discovery/v1"
	discoveryv1beta1 "k8s.io/client-go/kubernetes/typed/discovery/v1beta1"
	eventsv1 "k8s.io/client-go/kubernetes/typed/events/v1"
	eventsv1beta1 "k8s.io/client-go/kubernetes/typed/events/v1beta1"
	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	flowcontrolv1alpha1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1alpha1"
	flowcontrolv1beta1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta1"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	networkingv1beta1 "k8s.io/client-go/kubernetes/typed/networking/v1beta1"
	nodev1 "k8s.io/client-go/kubernetes/typed/node/v1"
	nodev1alpha1 "k8s.io/client-go/kubernetes/typed/node/v1alpha1"
	nodev1beta1 "k8s.io/client-go/kubernetes/typed/node/v1beta1"
	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"
	policyv1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	rbacv1alpha1 "k8s.io/client-go/kubernetes/typed/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	schedulingv1alpha1 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"
	schedulingv1beta1 "k8s.io/client-go/kubernetes/typed/scheduling/v1beta1"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	storagev1alpha1 "k8s.io/client-go/kubernetes/typed/storage/v1alpha1"
	storagev1beta1 "k8s.io/client-go/kubernetes/typed/storage/v1beta1"
)

type Clientset struct{}

func NewClientset() *Clientset {
	return &Clientset{}
}

// AdmissionregistrationV1 implements kubernetes.Interface
func (*Clientset) AdmissionregistrationV1() admissionregistrationv1.AdmissionregistrationV1Interface {
	panic("unimplemented")
}

// AdmissionregistrationV1beta1 implements kubernetes.Interface
func (*Clientset) AdmissionregistrationV1beta1() admissionregistrationv1beta1.AdmissionregistrationV1beta1Interface {
	panic("unimplemented")
}

// AppsV1 implements kubernetes.Interface
func (*Clientset) AppsV1() appsv1.AppsV1Interface {
	return NewAppV1()
}

// AppsV1beta1 implements kubernetes.Interface
func (*Clientset) AppsV1beta1() appsv1beta1.AppsV1beta1Interface {
	panic("unimplemented")
}

// AppsV1beta2 implements kubernetes.Interface
func (*Clientset) AppsV1beta2() appsv1beta2.AppsV1beta2Interface {
	panic("unimplemented")
}

// AuthenticationV1 implements kubernetes.Interface
func (*Clientset) AuthenticationV1() authenticationv1.AuthenticationV1Interface {
	panic("unimplemented")
}

// AuthenticationV1beta1 implements kubernetes.Interface
func (*Clientset) AuthenticationV1beta1() authenticationv1beta1.AuthenticationV1beta1Interface {
	panic("unimplemented")
}

// AuthorizationV1 implements kubernetes.Interface
func (*Clientset) AuthorizationV1() authorizationv1.AuthorizationV1Interface {
	panic("unimplemented")
}

// AuthorizationV1beta1 implements kubernetes.Interface
func (*Clientset) AuthorizationV1beta1() authorizationv1beta1.AuthorizationV1beta1Interface {
	panic("unimplemented")
}

// AutoscalingV1 implements kubernetes.Interface
func (*Clientset) AutoscalingV1() autoscalingv1.AutoscalingV1Interface {
	panic("unimplemented")
}

// AutoscalingV2beta1 implements kubernetes.Interface
func (*Clientset) AutoscalingV2beta1() autoscalingv2beta1.AutoscalingV2beta1Interface {
	panic("unimplemented")
}

// AutoscalingV2beta2 implements kubernetes.Interface
func (*Clientset) AutoscalingV2beta2() autoscalingv2beta2.AutoscalingV2beta2Interface {
	panic("unimplemented")
}

// BatchV1 implements kubernetes.Interface
func (*Clientset) BatchV1() batchv1.BatchV1Interface {
	panic("unimplemented")
}

// BatchV1beta1 implements kubernetes.Interface
func (*Clientset) BatchV1beta1() batchv1beta1.BatchV1beta1Interface {
	panic("unimplemented")
}

// CertificatesV1 implements kubernetes.Interface
func (*Clientset) CertificatesV1() certificatesv1.CertificatesV1Interface {
	panic("unimplemented")
}

// CertificatesV1beta1 implements kubernetes.Interface
func (*Clientset) CertificatesV1beta1() certificatesv1beta1.CertificatesV1beta1Interface {
	panic("unimplemented")
}

// CoordinationV1 implements kubernetes.Interface
func (*Clientset) CoordinationV1() coordinationv1.CoordinationV1Interface {
	panic("unimplemented")
}

// CoordinationV1beta1 implements kubernetes.Interface
func (*Clientset) CoordinationV1beta1() coordinationv1beta1.CoordinationV1beta1Interface {
	panic("unimplemented")
}

// CoreV1 implements kubernetes.Interface
func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return NewCoreV1()
}

// Discovery implements kubernetes.Interface
func (*Clientset) Discovery() discovery.DiscoveryInterface {
	panic("unimplemented")
}

// DiscoveryV1 implements kubernetes.Interface
func (*Clientset) DiscoveryV1() discoveryv1.DiscoveryV1Interface {
	panic("unimplemented")
}

// DiscoveryV1beta1 implements kubernetes.Interface
func (*Clientset) DiscoveryV1beta1() discoveryv1beta1.DiscoveryV1beta1Interface {
	panic("unimplemented")
}

// EventsV1 implements kubernetes.Interface
func (*Clientset) EventsV1() eventsv1.EventsV1Interface {
	panic("unimplemented")
}

// EventsV1beta1 implements kubernetes.Interface
func (*Clientset) EventsV1beta1() eventsv1beta1.EventsV1beta1Interface {
	panic("unimplemented")
}

// ExtensionsV1beta1 implements kubernetes.Interface
func (*Clientset) ExtensionsV1beta1() extensionsv1beta1.ExtensionsV1beta1Interface {
	panic("unimplemented")
}

// FlowcontrolV1alpha1 implements kubernetes.Interface
func (*Clientset) FlowcontrolV1alpha1() flowcontrolv1alpha1.FlowcontrolV1alpha1Interface {
	panic("unimplemented")
}

// FlowcontrolV1beta1 implements kubernetes.Interface
func (*Clientset) FlowcontrolV1beta1() flowcontrolv1beta1.FlowcontrolV1beta1Interface {
	panic("unimplemented")
}

// InternalV1alpha1 implements kubernetes.Interface
func (*Clientset) InternalV1alpha1() internalv1alpha1.InternalV1alpha1Interface {
	panic("unimplemented")
}

// NetworkingV1 implements kubernetes.Interface
func (*Clientset) NetworkingV1() networkingv1.NetworkingV1Interface {
	return NewNetworkingV1()
}

// NetworkingV1beta1 implements kubernetes.Interface
func (*Clientset) NetworkingV1beta1() networkingv1beta1.NetworkingV1beta1Interface {
	panic("unimplemented")
}

// NodeV1 implements kubernetes.Interface
func (*Clientset) NodeV1() nodev1.NodeV1Interface {
	panic("unimplemented")
}

// NodeV1alpha1 implements kubernetes.Interface
func (*Clientset) NodeV1alpha1() nodev1alpha1.NodeV1alpha1Interface {
	panic("unimplemented")
}

// NodeV1beta1 implements kubernetes.Interface
func (*Clientset) NodeV1beta1() nodev1beta1.NodeV1beta1Interface {
	panic("unimplemented")
}

// PolicyV1 implements kubernetes.Interface
func (*Clientset) PolicyV1() policyv1.PolicyV1Interface {
	panic("unimplemented")
}

// PolicyV1beta1 implements kubernetes.Interface
func (*Clientset) PolicyV1beta1() policyv1beta1.PolicyV1beta1Interface {
	panic("unimplemented")
}

// RbacV1 implements kubernetes.Interface
func (*Clientset) RbacV1() rbacv1.RbacV1Interface {
	panic("unimplemented")
}

// RbacV1alpha1 implements kubernetes.Interface
func (*Clientset) RbacV1alpha1() rbacv1alpha1.RbacV1alpha1Interface {
	panic("unimplemented")
}

// RbacV1beta1 implements kubernetes.Interface
func (*Clientset) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface {
	panic("unimplemented")
}

// SchedulingV1 implements kubernetes.Interface
func (*Clientset) SchedulingV1() schedulingv1.SchedulingV1Interface {
	panic("unimplemented")
}

// SchedulingV1alpha1 implements kubernetes.Interface
func (*Clientset) SchedulingV1alpha1() schedulingv1alpha1.SchedulingV1alpha1Interface {
	panic("unimplemented")
}

// SchedulingV1beta1 implements kubernetes.Interface
func (*Clientset) SchedulingV1beta1() schedulingv1beta1.SchedulingV1beta1Interface {
	panic("unimplemented")
}

// StorageV1 implements kubernetes.Interface
func (*Clientset) StorageV1() storagev1.StorageV1Interface {
	panic("unimplemented")
}

// StorageV1alpha1 implements kubernetes.Interface
func (*Clientset) StorageV1alpha1() storagev1alpha1.StorageV1alpha1Interface {
	panic("unimplemented")
}

// StorageV1beta1 implements kubernetes.Interface
func (*Clientset) StorageV1beta1() storagev1beta1.StorageV1beta1Interface {
	panic("unimplemented")
}
