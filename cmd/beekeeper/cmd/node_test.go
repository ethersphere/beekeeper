package cmd

import (
	"testing"

	"github.com/ethersphere/beekeeper/pkg/node"
)

func TestResolveDiscoveryType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		requested        node.DiscoveryType
		namespaceSet     bool
		clusterName      string
		statefulSetNames []string
		want             node.DiscoveryType
		wantErr          bool
	}{
		{name: "default with cluster name", clusterName: "default", want: node.DiscoveryTypeBeeClients},
		{name: "default with namespace", namespaceSet: true, want: node.DiscoveryTypeNamespace},
		{name: "default namespace overrides cluster name", namespaceSet: true, clusterName: "default", want: node.DiscoveryTypeNamespace},
		{name: "default ignores stateful sets", clusterName: "default", statefulSetNames: []string{"bee"}, want: node.DiscoveryTypeBeeClients},
		{name: "beeclients with cluster name", requested: node.DiscoveryTypeBeeClients, namespaceSet: true, clusterName: "default", want: node.DiscoveryTypeBeeClients},
		{name: "beeclients without cluster name", requested: node.DiscoveryTypeBeeClients, namespaceSet: true, wantErr: true},
		{name: "namespace with cluster name", requested: node.DiscoveryTypeNamespace, clusterName: "default", want: node.DiscoveryTypeNamespace},
		{name: "statefulset with names", requested: node.DiscoveryTypeStatefulSet, namespaceSet: true, statefulSetNames: []string{"bee"}, want: node.DiscoveryTypeStatefulSet},
		{name: "statefulset without names", requested: node.DiscoveryTypeStatefulSet, namespaceSet: true, wantErr: true},
		{name: "unsupported", requested: "pods", namespaceSet: true, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveDiscoveryType(test.requested, test.namespaceSet, test.clusterName, test.statefulSetNames)
			if (err != nil) != test.wantErr {
				t.Fatalf("expected error: %v, got: %v", test.wantErr, err)
			}
			if got != test.want {
				t.Errorf("expected discovery type %q, got %q", test.want, got)
			}
		})
	}
}
