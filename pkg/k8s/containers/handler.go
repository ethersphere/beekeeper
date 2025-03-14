package containers

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// LifecycleHandler represents Kubernetes LifecycleHandler
type LifecycleHandler struct {
	Exec      *ExecHandler
	HTTPGet   *HTTPGetHandler
	TCPSocket *TCPSocketHandler
}

// toK8S converts Handler to Kubernetes client object
func (h *LifecycleHandler) toK8S() v1.LifecycleHandler {
	if h.Exec != nil {
		return v1.LifecycleHandler{
			Exec: h.Exec.toK8S(),
		}
	} else if h.HTTPGet != nil {
		return v1.LifecycleHandler{
			HTTPGet: h.HTTPGet.toK8S(),
		}
	} else if h.TCPSocket != nil {
		return v1.LifecycleHandler{
			TCPSocket: h.TCPSocket.toK8S(),
		}
	}
	return v1.LifecycleHandler{}
}

// ExecHandler represents Kubernetes ExecAction Handler
type ExecHandler struct {
	Command []string
}

// toK8S converts ExecHandler to Kubernetes client object
func (eh *ExecHandler) toK8S() *v1.ExecAction {
	return &v1.ExecAction{
		Command: eh.Command,
	}
}

// HTTPGetHandler represents Kubernetes HTTPGetAction Handler
type HTTPGetHandler struct {
	Host        string
	Path        string
	Port        string
	Scheme      string
	HTTPHeaders HTTPHeaders
}

// toK8S converts HTTPGetHandler to Kubernetes client object
func (hg *HTTPGetHandler) toK8S() *v1.HTTPGetAction {
	return &v1.HTTPGetAction{
		Host:        hg.Host,
		Path:        hg.Path,
		Port:        intstr.FromString(hg.Port),
		Scheme:      v1.URIScheme(hg.Scheme),
		HTTPHeaders: hg.HTTPHeaders.toK8S(),
	}
}

// HTTPHeaders represents Kubernetes HTTPHeader
type HTTPHeaders []HTTPHeader

// toK8S converts HTTPHeaders to Kubernetes client objects
func (hhs HTTPHeaders) toK8S() (l []v1.HTTPHeader) {
	if len(hhs) > 0 {
		l = make([]v1.HTTPHeader, 0, len(hhs))
		for _, h := range hhs {
			l = append(l, h.toK8S())
		}
	}
	return
}

// HTTPHeader represents Kubernetes HTTPHeader
type HTTPHeader struct {
	Name  string
	Value string
}

// toK8S converts HTTPHeader to Kubernetes client object
func (hh *HTTPHeader) toK8S() v1.HTTPHeader {
	return v1.HTTPHeader{
		Name:  hh.Name,
		Value: hh.Value,
	}
}

// TCPSocketHandler represents Kubernetes TCPSocket Handler
type TCPSocketHandler struct {
	Host string
	Port string
}

// toK8S converts TCPSocketHandler to Kubernetes client object
func (tcps *TCPSocketHandler) toK8S() *v1.TCPSocketAction {
	return &v1.TCPSocketAction{
		Host: tcps.Host,
		Port: intstr.FromString(tcps.Port),
	}
}
