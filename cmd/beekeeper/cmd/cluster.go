package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/config"
	"golang.org/x/sync/errgroup"
)

func (c *command) deleteCluster(ctx context.Context, clusterName string, cfg *config.Config, deleteStorage bool) (err error) {
	clusterConfig, ok := cfg.Clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s not defined", clusterName)
	}

	clusterOptions := clusterConfig.Export()
	clusterOptions.K8SClient = c.k8sClient
	clusterOptions.SwapClient = c.swapClient

	cluster := bee.NewCluster(clusterConfig.GetName(), clusterOptions)

	// delete node groups
	for ng, v := range clusterConfig.GetNodeGroups() {
		fmt.Printf("deleting %s node group\n", ng)
		ngConfig, ok := cfg.NodeGroups[v.Config]
		if !ok {
			return fmt.Errorf("node group profile %s not defined", v.Config)
		}
		if v.Mode == "bootnode" { // TODO: implement standalone mode
			// register node group
			cluster.AddNodeGroup(ng, ngConfig.Export())

			// delete nodes from the node group
			g := cluster.NodeGroup(ng)
			for i := 0; i < len(v.Nodes); i++ {
				nName := v.Nodes[i].Name
				if err := g.DeleteNode(ctx, nName); err != nil {
					return fmt.Errorf("deleting node %s from the node group %s", nName, ng)
				}

				if deleteStorage && *ngConfig.PersistenceEnabled {
					pvcName := fmt.Sprintf("data-%s-0", nName)
					if err := c.k8sClient.PVC.Delete(ctx, pvcName, clusterOptions.Namespace); err != nil {
						return fmt.Errorf("deleting pvc %s: %w", pvcName, err)
					}
				}
			}
		} else {
			// register node group
			cluster.AddNodeGroup(ng, ngConfig.Export())

			// delete nodes from the node group
			g := cluster.NodeGroup(ng)
			for i := 0; i < v.Count; i++ {
				nName := fmt.Sprintf("%s-%d", ng, i)
				if err := g.DeleteNode(ctx, nName); err != nil {
					return fmt.Errorf("deleting node %s from the node group %s", nName, ng)
				}

				if deleteStorage && *ngConfig.PersistenceEnabled {
					pvcName := fmt.Sprintf("data-%s-0", nName)
					if err := c.k8sClient.PVC.Delete(ctx, pvcName, clusterOptions.Namespace); err != nil {
						return fmt.Errorf("deleting pvc %s: %w", pvcName, err)
					}
				}
			}
		}
	}

	return
}

func (c *command) setupCluster(ctx context.Context, clusterName string, cfg *config.Config, start bool) (cluster *bee.Cluster, err error) {
	clusterConfig, ok := cfg.Clusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("cluster %s not defined", clusterName)
	}

	clusterOptions := clusterConfig.Export()
	clusterOptions.K8SClient = c.k8sClient
	clusterOptions.SwapClient = c.swapClient

	cluster = bee.NewCluster(clusterConfig.GetName(), clusterOptions)

	if start {
		bootnodes := ""
		for ng, v := range clusterConfig.GetNodeGroups() {
			ngConfig, ok := cfg.NodeGroups[v.Config]
			if !ok {
				return nil, fmt.Errorf("node group profile %s not defined", v.Config)
			}
			if v.Mode == "bootnode" { // TODO: implement standalone mode
				// add node group to the cluster
				cluster.AddNodeGroup(ng, ngConfig.Export())

				// start nodes in the node group
				g := cluster.NodeGroup(ng)
				errGroup := new(errgroup.Group)
				for i := 0; i < len(v.Nodes); i++ {
					nName := v.Nodes[i].Name
					beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
					if !ok {
						return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
					}
					bConfig := beeConfig.Export()

					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, clusterConfig.GetNamespace()) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					bOptions := bee.NodeOptions{
						Config:       &bConfig,
						ClefKey:      v.Nodes[i].ClefKey,
						ClefPassword: v.Nodes[i].ClefPassword,
						LibP2PKey:    v.Nodes[i].LibP2PKey,
						SwarmKey:     v.Nodes[i].SwarmKey,
					}

					errGroup.Go(func() error {
						return g.SetupNode(ctx, nName, bOptions)
					})
				}

				if err := errGroup.Wait(); err != nil {
					return nil, fmt.Errorf("starting node group %s: %w", ng, err)
				}
			}
		}

		for ng, v := range clusterConfig.GetNodeGroups() {
			ngConfig, ok := cfg.NodeGroups[v.Config]
			if !ok {
				return nil, fmt.Errorf("node group profile %s not defined", v.Config)
			}
			if v.Mode != "bootnode" { // TODO: support standalone nodes
				// add node group to the cluster
				gOptions := ngConfig.Export()
				nProfile, ok := cfg.BeeConfigs[v.BeeConfig]
				if !ok {
					return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
				}
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
						return g.SetupNode(ctx, nName, bee.NodeOptions{})
					})
				}

				if err := errGroup.Wait(); err != nil {
					return nil, fmt.Errorf("starting node group %s: %w", ng, err)
				}
			}
		}
	} else {
		bootnodes := ""
		for ng, v := range clusterConfig.GetNodeGroups() {
			ngConfig, ok := cfg.NodeGroups[v.Config]
			if !ok {
				return nil, fmt.Errorf("node group profile %s not defined", v.Config)
			}
			if v.Mode == "bootnode" { // TODO: support standalone nodes
				// add node group to the cluster
				cluster.AddNodeGroup(ng, ngConfig.Export())

				// add nodes to the node group
				g := cluster.NodeGroup(ng)
				for i := 0; i < len(v.Nodes); i++ {
					nName := v.Nodes[i].Name
					beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
					if !ok {
						return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
					}
					bConfig := beeConfig.Export()

					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, clusterConfig.GetNamespace()) // TODO: improve bootnode management, support more than 2 bootnodes
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

		for ng, v := range clusterConfig.GetNodeGroups() {
			ngConfig, ok := cfg.NodeGroups[v.Config]
			if !ok {
				return nil, fmt.Errorf("node group profile %s not defined", v.Config)
			}
			if v.Mode != "bootnode" { // TODO: support standalone nodes
				// add node group to the cluster
				gOptions := ngConfig.Export()
				nProfile, ok := cfg.BeeConfigs[v.BeeConfig]
				if !ok {
					return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
				}
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
