package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Handler ...
type Handler struct {
	Exec      *ExecHandler
	HTTPGet   *HTTPGetHandler
	TCPSocket *TCPSocketHandler
}

func (h Handler) toK8S() v1.Handler {
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

// ExecHandler ...
type ExecHandler struct {
	Command []string
}

func (eh ExecHandler) toK8S() v1.Handler {
	return v1.Handler{
		Exec: &v1.ExecAction{
			Command: eh.Command,
		},
	}
}

// HTTPGetHandler ...
type HTTPGetHandler struct {
	Host        string
	Path        string
	Port        string
	Scheme      string
	HTTPHeaders []HTTPHeader
}

func (hg HTTPGetHandler) toK8S() v1.Handler {
	return v1.Handler{
		HTTPGet: &v1.HTTPGetAction{
			Host:        hg.Host,
			Path:        hg.Path,
			Port:        intstr.FromString(hg.Port),
			Scheme:      v1.URIScheme(hg.Scheme),
			HTTPHeaders: httpHeaderToK8S(hg.HTTPHeaders),
		},
	}
}

// HTTPHeader ...
type HTTPHeader struct {
	Name  string
	Value string
}

func (hh HTTPHeader) toK8S() v1.HTTPHeader {
	return v1.HTTPHeader{
		Name:  hh.Name,
		Value: hh.Value,
	}
}

func httpHeaderToK8S(headers []HTTPHeader) (l []v1.HTTPHeader) {
	for _, header := range headers {
		l = append(l, header.toK8S())
	}
	return
}

// TCPSocketHandler ...
type TCPSocketHandler struct {
	Host string
	Port string
}

func (tcps TCPSocketHandler) toK8S() v1.Handler {
	return v1.Handler{
		TCPSocket: &v1.TCPSocketAction{
			Host: tcps.Host,
			Port: intstr.FromString(tcps.Port),
		},
	}
}
