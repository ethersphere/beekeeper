package containers

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Handler represents Kubernetes Handler
type Handler struct {
	Exec      *ExecHandler
	HTTPGet   *HTTPGetHandler
	TCPSocket *TCPSocketHandler
}

// toK8S converts Handler to Kuberntes client object
func (h *Handler) toK8S() v1.Handler {
	if h.Exec != nil {
		return h.Exec.toK8S()
	} else if h.HTTPGet != nil {
		return h.HTTPGet.toK8S()
	} else if h.TCPSocket != nil {
		return h.TCPSocket.toK8S()
	} else {
		return v1.Handler{}
	}
}

// ExecHandler represents Kubernetes ExecAction Handler
type ExecHandler struct {
	Command []string
}

// toK8S converts ExecHandler to Kuberntes client object
func (eh *ExecHandler) toK8S() v1.Handler {
	return v1.Handler{
		Exec: &v1.ExecAction{
			Command: eh.Command,
		},
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

// toK8S converts HTTPGetHandler to Kuberntes client object
func (hg *HTTPGetHandler) toK8S() v1.Handler {
	return v1.Handler{
		HTTPGet: &v1.HTTPGetAction{
			Host:        hg.Host,
			Path:        hg.Path,
			Port:        intstr.FromString(hg.Port),
			Scheme:      v1.URIScheme(hg.Scheme),
			HTTPHeaders: hg.HTTPHeaders.toK8S(),
		},
	}
}

// HTTPHeaders represents Kubernetes HTTPHeader
type HTTPHeaders []HTTPHeader

// toK8S converts HTTPHeaders to Kuberntes client objects
func (hhs *HTTPHeaders) toK8S() (l []v1.HTTPHeader) {
	l = make([]v1.HTTPHeader, 0, len(*hhs))

	for _, h := range *hhs {
		l = append(l, h.toK8S())
	}

	return
}

// HTTPHeader represents Kubernetes HTTPHeader
type HTTPHeader struct {
	Name  string
	Value string
}

// toK8S converts HTTPHeader to Kuberntes client object
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

// toK8S converts TCPSocketHandler to Kuberntes client object
func (tcps *TCPSocketHandler) toK8S() v1.Handler {
	return v1.Handler{
		TCPSocket: &v1.TCPSocketAction{
			Host: tcps.Host,
			Port: intstr.FromString(tcps.Port),
		},
	}
}
