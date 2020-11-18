package containers

import (
	v1 "k8s.io/api/core/v1"
)

// Probe represents Kubernetes Probe
type Probe struct {
	Exec      *ExecProbe
	HTTPGet   *HTTPGetProbe
	TCPSocket *TCPSocketProbe
}

// toK8S converts Containers to Kuberntes client object
func (p *Probe) toK8S() *v1.Probe {
	if p.Exec != nil {
		return p.Exec.toK8S()
	} else if p.HTTPGet != nil {
		return p.HTTPGet.toK8S()
	} else if p.TCPSocket != nil {
		return p.TCPSocket.toK8S()
	} else {
		return nil
	}
}

// ExecProbe represents Kubernetes ExecHandler Probe
type ExecProbe struct {
	FailureThreshold    int32
	Handler             ExecHandler
	InitialDelaySeconds int32
	PeriodSeconds       int32
	SuccessThreshold    int32
	TimeoutSeconds      int32
}

// toK8S converts ExecProbe to Kuberntes client object
func (ep *ExecProbe) toK8S() *v1.Probe {
	return &v1.Probe{
		FailureThreshold:    ep.FailureThreshold,
		Handler:             ep.Handler.toK8S(),
		InitialDelaySeconds: ep.InitialDelaySeconds,
		PeriodSeconds:       ep.PeriodSeconds,
		SuccessThreshold:    ep.SuccessThreshold,
		TimeoutSeconds:      ep.TimeoutSeconds,
	}
}

// HTTPGetProbe represents Kubernetes HTTPGetHandler Probe
type HTTPGetProbe struct {
	FailureThreshold    int32
	Handler             HTTPGetHandler
	InitialDelaySeconds int32
	PeriodSeconds       int32
	SuccessThreshold    int32
	TimeoutSeconds      int32
}

// toK8S converts HTTPGetProbe to Kuberntes client object
func (hgp *HTTPGetProbe) toK8S() *v1.Probe {
	return &v1.Probe{
		FailureThreshold:    hgp.FailureThreshold,
		Handler:             hgp.Handler.toK8S(),
		InitialDelaySeconds: hgp.InitialDelaySeconds,
		PeriodSeconds:       hgp.PeriodSeconds,
		SuccessThreshold:    hgp.SuccessThreshold,
		TimeoutSeconds:      hgp.TimeoutSeconds,
	}
}

// TCPSocketProbe represents Kubernetes TCPSocketHandler Probe
type TCPSocketProbe struct {
	FailureThreshold    int32
	Handler             TCPSocketHandler
	InitialDelaySeconds int32
	PeriodSeconds       int32
	SuccessThreshold    int32
	TimeoutSeconds      int32
}

// toK8S converts TCPSocketProbe to Kuberntes client object
func (tsp *TCPSocketProbe) toK8S() *v1.Probe {
	return &v1.Probe{
		FailureThreshold:    tsp.FailureThreshold,
		Handler:             tsp.Handler.toK8S(),
		InitialDelaySeconds: tsp.InitialDelaySeconds,
		PeriodSeconds:       tsp.PeriodSeconds,
		SuccessThreshold:    tsp.SuccessThreshold,
		TimeoutSeconds:      tsp.TimeoutSeconds,
	}
}
