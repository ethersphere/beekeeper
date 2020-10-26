package statefulset

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Resources ...
type Resources struct {
	Limit   Limit
	Request Request
}

func (r Resources) toK8S() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Limits:   r.Limit.toK8S(),
		Requests: r.Request.toK8S(),
	}
}

// Limit ...
type Limit struct {
	CPU              string
	Memory           string
	Storage          string
	EphemeralStorage string
}

func (l Limit) toK8S() v1.ResourceList {
	m := map[v1.ResourceName]resource.Quantity{}
	if len(l.CPU) > 0 {
		m[v1.ResourceCPU] = resource.MustParse(l.CPU)
	}
	if len(l.Memory) > 0 {
		m[v1.ResourceMemory] = resource.MustParse(l.Memory)
	}
	if len(l.Storage) > 0 {
		m[v1.ResourceStorage] = resource.MustParse(l.Storage)
	}
	if len(l.EphemeralStorage) > 0 {
		m[v1.ResourceEphemeralStorage] = resource.MustParse(l.EphemeralStorage)
	}
	return m
}

// Request ...
type Request struct {
	CPU              string
	Memory           string
	Storage          string
	EphemeralStorage string
}

func (r Request) toK8S() v1.ResourceList {
	m := map[v1.ResourceName]resource.Quantity{}
	if len(r.CPU) > 0 {
		m[v1.ResourceCPU] = resource.MustParse(r.CPU)
	}
	if len(r.Memory) > 0 {
		m[v1.ResourceMemory] = resource.MustParse(r.Memory)
	}
	if len(r.Storage) > 0 {
		m[v1.ResourceStorage] = resource.MustParse(r.Storage)
	}
	if len(r.EphemeralStorage) > 0 {
		m[v1.ResourceEphemeralStorage] = resource.MustParse(r.EphemeralStorage)
	}
	return m
}
