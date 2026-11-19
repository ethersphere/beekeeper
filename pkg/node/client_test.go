package node_test

import (
	"errors"
	"io"
	"testing"

	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute"
	irmock "github.com/ethersphere/beekeeper/pkg/k8s/customresource/ingressroute/mock"
	"github.com/ethersphere/beekeeper/pkg/k8s/ingress"
	"github.com/ethersphere/beekeeper/pkg/k8s/pod"
	"github.com/ethersphere/beekeeper/pkg/k8s/service"
	"github.com/ethersphere/beekeeper/pkg/k8s/statefulset"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/node"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = "test"
	instanceLabel = "app.kubernetes.io/instance"
)

func newK8sClient(objects ...runtime.Object) *k8s.Client {
	log := logging.New(io.Discard, 0)
	clientset := fake.NewClientset(objects...)

	return &k8s.Client{
		Ingress:      ingress.NewClient(clientset, log),
		IngressRoute: ingressroute.NewClient(irmock.New(), log),
		Pods:         pod.NewClient(clientset, log),
		Service:      service.NewClient(clientset),
		StatefulSet:  statefulset.NewClient(clientset, log),
	}
}

func newStatefulSet(name, instance string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels:    map[string]string{instanceLabel: instance},
		},
		Spec: appsv1.StatefulSetSpec{Replicas: &replicas},
	}
}

func newPod(name, ip string, apiPort int32) *v1.Pod {
	p := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Status:     v1.PodStatus{PodIP: ip},
	}
	if apiPort != 0 {
		p.Spec.Containers = []v1.Container{{
			Name:  "bee",
			Ports: []v1.ContainerPort{{Name: "api", ContainerPort: apiPort}},
		}}
	}
	return p
}

func newService(name, instance, clusterIP string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels:    map[string]string{instanceLabel: instance},
		},
		Spec: v1.ServiceSpec{
			ClusterIP: clusterIP,
			Ports:     []v1.ServicePort{{Name: "api", Port: 1633}},
		},
	}
}

func newIngress(name, instance, host string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels:    map[string]string{instanceLabel: instance},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{Host: host}},
		},
	}
}

func TestGetNodes_StatefulSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		objects          []runtime.Object
		statefulSetNames []string
		inCluster        bool
		want             map[string]string // node name -> endpoint
		wantErr          error
	}{
		{
			// Beekeeper deployment: one StatefulSet with a single replica per
			// node, the api service is named after the StatefulSet.
			name: "beekeeper in cluster uses service named after statefulset",
			objects: []runtime.Object{
				newStatefulSet("bee-0", "bee-0", 1),
				newPod("bee-0-0", "10.0.0.1", 1633),
				newService("bee-0", "bee-0", "10.1.0.1"),
				newStatefulSet("bee-1", "bee-1", 1),
				newPod("bee-1-0", "10.0.0.2", 1633),
				newService("bee-1", "bee-1", "10.1.0.2"),
			},
			statefulSetNames: []string{"bee-1", "bee-0", "bee-1"},
			inCluster:        true,
			want: map[string]string{
				"bee-0-0": "http://10.1.0.1:1633",
				"bee-1-0": "http://10.1.0.2:1633",
			},
		},
		{
			name: "beekeeper out of cluster uses ingress named after statefulset",
			objects: []runtime.Object{
				newStatefulSet("bee-0", "bee-0", 1),
				newPod("bee-0-0", "10.0.0.1", 1633),
				newService("bee-0", "bee-0", "10.1.0.1"),
				newIngress("bee-0", "bee-0", "bee-0.localhost"),
			},
			statefulSetNames: []string{"bee-0"},
			want: map[string]string{
				"bee-0-0": "http://bee-0.localhost",
			},
		},
		{
			// Helm deployment: one StatefulSet with many replicas, resources
			// of all replicas share the instance label and are named after
			// the pods.
			name: "helm resolves an endpoint per replica",
			objects: []runtime.Object{
				newStatefulSet("swarm-bee", "swarm", 2),
				newPod("swarm-bee-0", "10.0.0.1", 1633),
				newPod("swarm-bee-1", "10.0.0.2", 1633),
				newIngress("swarm-bee-0", "swarm", "bee-0.example.com"),
				newIngress("swarm-bee-1", "swarm", "bee-1.example.com"),
			},
			statefulSetNames: []string{"swarm-bee"},
			want: map[string]string{
				"swarm-bee-0": "http://bee-0.example.com",
				"swarm-bee-1": "http://bee-1.example.com",
			},
		},
		{
			name: "shared service is not used for multiple replicas",
			objects: []runtime.Object{
				newStatefulSet("bee", "bee", 2),
				newPod("bee-0", "10.0.0.1", 1633),
				newPod("bee-1", "10.0.0.2", 1633),
				newService("bee", "bee", "10.1.0.1"),
			},
			statefulSetNames: []string{"bee"},
			inCluster:        true,
			want: map[string]string{
				"bee-0": "http://10.0.0.1:1633",
				"bee-1": "http://10.0.0.2:1633",
			},
		},
		{
			name: "falls back to pod IP and api container port",
			objects: []runtime.Object{
				newStatefulSet("bootnode-0", "bootnode-0", 1),
				newPod("bootnode-0-0", "10.0.0.1", 8080),
				newStatefulSet("bootnode-1", "bootnode-1", 1),
				newPod("bootnode-1-0", "10.0.0.2", 0),
			},
			statefulSetNames: []string{"bootnode-0", "bootnode-1"},
			want: map[string]string{
				"bootnode-0-0": "http://10.0.0.1:8080",
				"bootnode-1-0": "http://10.0.0.2:1633",
			},
		},
		{
			name: "missing statefulset and pods are skipped",
			objects: []runtime.Object{
				newStatefulSet("bee", "bee", 3),
				newPod("bee-0", "10.0.0.1", 1633),
				newPod("bee-2", "", 1633), // no IP and no service
			},
			statefulSetNames: []string{"missing", "bee"},
			inCluster:        true,
			want: map[string]string{
				"bee-0": "http://10.0.0.1:1633",
			},
		},
		{
			name: "no nodes found",
			objects: []runtime.Object{
				newStatefulSet("bee", "bee", 0),
			},
			statefulSetNames: []string{"bee", "missing"},
			wantErr:          node.ErrNodesNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := node.New(&node.ClientConfig{
				K8sClient:        newK8sClient(test.objects...),
				Namespace:        testNamespace,
				InCluster:        test.inCluster,
				DiscoveryType:    node.DiscoveryTypeStatefulSet,
				StatefulSetNames: test.statefulSetNames,
			})

			nodes, err := client.GetNodes(t.Context())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got: %v", test.wantErr, err)
			}

			if len(nodes) != len(test.want) {
				t.Fatalf("expected %d nodes, got %d: %v", len(test.want), len(nodes), nodes)
			}

			for i, n := range nodes {
				want, ok := test.want[n.Name()]
				if !ok {
					t.Errorf("unexpected node %s", n.Name())
					continue
				}
				if got := n.Client().BaseURL().String(); got != want {
					t.Errorf("node %s: expected endpoint %s, got %s", n.Name(), want, got)
				}
				if i > 0 && nodes[i-1].Name() > n.Name() {
					t.Errorf("nodes are not sorted: %s before %s", nodes[i-1].Name(), n.Name())
				}
			}
		})
	}
}

func TestGetNodes_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  node.ClientConfig
	}{
		{
			name: "unknown discovery type",
			cfg:  node.ClientConfig{Namespace: testNamespace},
		},
		{
			name: "beeclients without clients",
			cfg:  node.ClientConfig{DiscoveryType: node.DiscoveryTypeBeeClients},
		},
		{
			name: "namespace without namespace",
			cfg:  node.ClientConfig{DiscoveryType: node.DiscoveryTypeNamespace, K8sClient: newK8sClient()},
		},
		{
			name: "namespace without k8s client",
			cfg:  node.ClientConfig{DiscoveryType: node.DiscoveryTypeNamespace, Namespace: testNamespace},
		},
		{
			name: "statefulset without k8s client",
			cfg:  node.ClientConfig{DiscoveryType: node.DiscoveryTypeStatefulSet, Namespace: testNamespace, StatefulSetNames: []string{"bee"}},
		},
		{
			name: "statefulset without namespace",
			cfg:  node.ClientConfig{DiscoveryType: node.DiscoveryTypeStatefulSet, K8sClient: newK8sClient(), StatefulSetNames: []string{"bee"}},
		},
		{
			name: "statefulset without names",
			cfg:  node.ClientConfig{DiscoveryType: node.DiscoveryTypeStatefulSet, K8sClient: newK8sClient(), Namespace: testNamespace},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := node.New(&test.cfg).GetNodes(t.Context()); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestGetNodes_Namespace(t *testing.T) {
	t.Parallel()

	client := node.New(&node.ClientConfig{
		K8sClient: newK8sClient(
			newService("bee-0", "bee-0", "10.1.0.1"),
			newService("bee-1", "bee-1", "10.1.0.2"),
		),
		Namespace:      testNamespace,
		InCluster:      true,
		DeploymentType: node.DeploymentTypeBeekeeper,
		DiscoveryType:  node.DiscoveryTypeNamespace,
	})

	nodes, err := client.GetNodes(t.Context())
	if err != nil {
		t.Fatalf("error not expected, got: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	for _, name := range []string{"bee-0-0", "bee-1-0"} {
		if nodes.Get(name) == nil {
			t.Errorf("expected node %s", name)
		}
	}
}
