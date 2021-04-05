package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/stress"
)

// newDefaultNodeGroupOptions returns default node group options
func newDefaultNodeGroupOptions() *bee.NodeGroupOptions {
	return &bee.NodeGroupOptions{
		ClefImage:           "ethersphere/clef:latest",
		ClefImagePullPolicy: "IfNotPresent",
		Image:               "ethersphere/bee:latest",
		ImagePullPolicy:     "IfNotPresent",
		IngressAnnotations: map[string]string{
			"kubernetes.io/ingress.class":                        "nginx-internal",
			"nginx.ingress.kubernetes.io/affinity":               "cookie",
			"nginx.ingress.kubernetes.io/affinity-mode":          "persistent",
			"nginx.ingress.kubernetes.io/proxy-body-size":        "0",
			"nginx.ingress.kubernetes.io/proxy-read-timeout":     "7200",
			"nginx.ingress.kubernetes.io/proxy-send-timeout":     "7200",
			"nginx.ingress.kubernetes.io/session-cookie-max-age": "86400",
			"nginx.ingress.kubernetes.io/session-cookie-name":    "SWARMGATEWAY",
			"nginx.ingress.kubernetes.io/session-cookie-path":    "default",
			"nginx.ingress.kubernetes.io/ssl-redirect":           "true",
		},
		IngressDebugAnnotations: map[string]string{
			"kubernetes.io/ingress.class": "nginx-internal",
		},
		Labels: map[string]string{
			"app.kubernetes.io/component": "bee",
			"app.kubernetes.io/part-of":   "bee",
			"app.kubernetes.io/version":   "latest",
		},
		LimitCPU:    "1",
		LimitMemory: "2Gi",
		NodeSelector: map[string]string{
			"node-group": "bee-staging",
		},
		PersistenceEnabled:        true,
		PersistenceStorageClass:   "local-storage",
		PersistenceStorageRequest: "34Gi",
		PodManagementPolicy:       "OrderedReady",
		RestartPolicy:             "Always",
		RequestCPU:                "750m",
		RequestMemory:             "1Gi",
		UpdateStrategy:            "RollingUpdate",
	}
}

func setK8SClient(kubeconfig string, inCluster bool) (c *k8s.Client, err error) {
	if c, err = k8s.NewClient(&k8s.ClientOptions{
		InCluster:      inCluster,
		KubeconfigPath: kubeconfig,
	}); err != nil && err != k8s.ErrKubeconfigNotSet {
		return nil, fmt.Errorf("creating Kubernetes client: %w", err)
	}

	return c, nil
}

var checkStages = []check.Stage{
	[]check.Update{
		{
			NodeGroup: "bee",
			Actions: check.Actions{
				AddCount:    2,
				StartCount:  0,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: check.Actions{
				AddCount:    4,
				StartCount:  0,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
	},
	[]check.Update{
		{
			NodeGroup: "bee",
			Actions: check.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: check.Actions{
				AddCount:    2,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
	[]check.Update{
		{
			NodeGroup: "bee",
			Actions: check.Actions{
				AddCount:    4,
				StartCount:  1,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
		{
			NodeGroup: "drone",
			Actions: check.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
}

var stressStages = []stress.Stage{
	[]stress.Update{
		{
			NodeGroup: "bee",
			Actions: stress.Actions{
				AddCount:    2,
				StartCount:  0,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: stress.Actions{
				AddCount:    4,
				StartCount:  0,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
	},
	[]stress.Update{
		{
			NodeGroup: "bee",
			Actions: stress.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: stress.Actions{
				AddCount:    2,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
	[]stress.Update{
		{
			NodeGroup: "bee",
			Actions: stress.Actions{
				AddCount:    4,
				StartCount:  1,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
		{
			NodeGroup: "drone",
			Actions: stress.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
}
