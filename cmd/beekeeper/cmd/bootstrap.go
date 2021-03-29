package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/config"
	"golang.org/x/sync/errgroup"
)

func setupCluster(ctx context.Context, c *config.Config, start bool) (cluster *bee.Cluster, err error) {
	k8sClient, err := setK8SClient(c.Kubernetes.Kubeconfig, c.Kubernetes.InCluster)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}

	cluster = bee.NewCluster(c.Cluster.Name, bee.ClusterOptions{
		APIDomain:           c.Cluster.API.Domain,
		APIInsecureTLS:      c.Cluster.API.InsecureTLS,
		APIScheme:           c.Cluster.API.Scheme,
		DebugAPIDomain:      c.Cluster.DebugAPI.Domain,
		DebugAPIInsecureTLS: c.Cluster.DebugAPI.InsecureTLS,
		DebugAPIScheme:      c.Cluster.DebugAPI.Scheme,
		K8SClient:           k8sClient,
		Namespace:           c.Cluster.Namespace,
		DisableNamespace:    c.Cluster.DisableNamespace,
	})

	if start {
		bootnodes := ""
		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode == "bootnode" {
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				cluster.AddNodeGroup(ng, gProfile.Export())

				// start nodes in the node group
				g := cluster.NodeGroup(ng)
				errGroup := new(errgroup.Group)
				for i := 0; i < len(v.Nodes); i++ {
					nName := v.Nodes[i].Name
					bProfile := c.BeeProfiles[v.BeeConfig]
					bConfig := bProfile.Export()

					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, c.Cluster.Namespace) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					bOptions := bee.NodeOptions{
						Config:       &bConfig,
						ClefKey:      v.Nodes[i].ClefKey,
						ClefPassword: v.Nodes[i].ClefPassword,
						LibP2PKey:    v.Nodes[i].LibP2PKey,
						SwarmKey:     v.Nodes[i].SwarmKey,
					}

					errGroup.Go(func() error {
						return g.AddStartNode(ctx, nName, bOptions)
					})
				}

				if err := errGroup.Wait(); err != nil {
					return nil, fmt.Errorf("starting node group %s: %w", ng, err)
				}
			}
		}
		fmt.Println(1)
		for ng, v := range c.Cluster.NodeGroups {
			fmt.Println(2)
			if v.Mode != "bootnode" {
				fmt.Println(3)
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				gOptions := gProfile.Export()
				nProfile := c.BeeProfiles[v.BeeConfig]
				nConfig := nProfile.Export()
				nConfig.Bootnodes = bootnodes
				gOptions.BeeConfig = &nConfig
				cluster.AddNodeGroup(ng, gOptions)

				// start nodes in the node group
				g := cluster.NodeGroup(ng)
				errGroup := new(errgroup.Group)
				for i := 0; i < v.Count; i++ {
					nName := fmt.Sprintf("%s-%d", ng, i)

					errGroup.Go(func() error {
						return g.AddStartNode(ctx, nName, bee.NodeOptions{})
					})
				}

				if err := errGroup.Wait(); err != nil {
					return nil, fmt.Errorf("starting node group %s: %w", ng, err)
				}
			}
		}
	} else {
		fmt.Println("add")
		bootnodes := ""
		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode == "bootnode" {
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				cluster.AddNodeGroup(ng, gProfile.Export())

				// add nodes to the node group
				g := cluster.NodeGroup(ng)
				for i := 0; i < len(v.Nodes); i++ {
					nName := v.Nodes[i].Name
					bProfile := c.BeeProfiles[v.BeeConfig]
					bConfig := bProfile.Export()

					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, c.Cluster.Namespace) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					bOptions := bee.NodeOptions{
						Config:       &bConfig,
						ClefKey:      v.Nodes[i].ClefKey,
						ClefPassword: v.Nodes[i].ClefPassword,
						LibP2PKey:    v.Nodes[i].LibP2PKey,
						SwarmKey:     v.Nodes[i].SwarmKey,
					}

					if err := g.AddNode(nName, bOptions); err != nil {
						return nil, fmt.Errorf("adding node %s: %w", nName, err)
					}
				}
			}
		}

		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode != "bootnode" {
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				gOptions := gProfile.Export()
				nProfile := c.BeeProfiles[v.BeeConfig]
				nConfig := nProfile.Export()
				nConfig.Bootnodes = bootnodes
				gOptions.BeeConfig = &nConfig
				cluster.AddNodeGroup(ng, gOptions)

				// add nodes to the node group
				g := cluster.NodeGroup(ng)
				for i := 0; i < v.Count; i++ {
					nName := fmt.Sprintf("%s-%d", ng, i)

					if err := g.AddNode(nName, bee.NodeOptions{}); err != nil {
						return nil, fmt.Errorf("adding node %s: %w", nName, err)
					}
				}
			}
		}
	}

	return
}
