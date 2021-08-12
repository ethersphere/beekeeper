package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	orchestrationK8S "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
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

	cluster := orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterOptions)

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
			g, err := cluster.NodeGroup(ng)
			if err != nil {
				return err
			}
			for i := 0; i < len(v.Nodes); i++ {
				nName := fmt.Sprintf("%s-%d", ng, i)
				if len(v.Nodes[i].Name) > 0 {
					nName = v.Nodes[i].Name
				}
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
			g, err := cluster.NodeGroup(ng)
			if err != nil {
				return err
			}
			if len(v.Nodes) > 0 {
				for i := 0; i < len(v.Nodes); i++ {
					nName := fmt.Sprintf("%s-%d", ng, i)
					if len(v.Nodes[i].Name) > 0 {
						nName = v.Nodes[i].Name
					}
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
	}

	return
}

func (c *command) setupCluster(ctx context.Context, clusterName string, cfg *config.Config, start bool) (cluster orchestration.Cluster, err error) {
	clusterConfig, ok := cfg.Clusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("cluster %s not defined", clusterName)
	}

	clusterOptions := clusterConfig.Export()
	clusterOptions.K8SClient = c.k8sClient
	clusterOptions.SwapClient = c.swapClient

	cluster = orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterOptions)

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
				g, err := cluster.NodeGroup(ng)
				if err != nil {
					return nil, err
				}
				errGroup := new(errgroup.Group)
				for i := 0; i < len(v.Nodes); i++ {
					// set node name
					nName := fmt.Sprintf("%s-%d", ng, i)
					if len(v.Nodes[i].Name) > 0 {
						nName = v.Nodes[i].Name
					}
					// set bootnodes
					beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
					if !ok {
						return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
					}
					bConfig := beeConfig.Export()
					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, clusterConfig.GetNamespace()) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					// set NodeOptions
					nOptions := orchestration.NodeOptions{
						Config: &bConfig,
					}
					if len(v.Nodes[i].Clef.Key) > 0 {
						nOptions.ClefKey = v.Nodes[i].Clef.Key
					}
					if len(v.Nodes[i].Clef.Password) > 0 {
						nOptions.ClefPassword = v.Nodes[i].Clef.Password
					}
					if len(v.Nodes[i].LibP2PKey) > 0 {
						nOptions.LibP2PKey = v.Nodes[i].LibP2PKey
					}
					if len(v.Nodes[i].SwarmKey) > 0 {
						nOptions.SwarmKey = v.Nodes[i].SwarmKey
					}

					errGroup.Go(func() error {
						return g.SetupNode(ctx, nName, nOptions, clusterConfig.Funding.Export())
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
				// set bootnodes
				beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
				if !ok {
					return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
				}
				bConfig := beeConfig.Export()
				bConfig.Bootnodes = bootnodes
				// add node group to the cluster
				ngOptions := ngConfig.Export()
				ngOptions.BeeConfig = &bConfig
				cluster.AddNodeGroup(ng, ngOptions)

				// start nodes in the node group
				g, err := cluster.NodeGroup(ng)
				if err != nil {
					return nil, err
				}
				errGroup := new(errgroup.Group)

				if len(v.Nodes) > 0 {
					for i := 0; i < len(v.Nodes); i++ {
						// set node name
						nName := fmt.Sprintf("%s-%d", ng, i)
						if len(v.Nodes[i].Name) > 0 {
							nName = v.Nodes[i].Name
						}
						// set NodeOptions
						nOptions := orchestration.NodeOptions{}
						if len(v.Nodes[i].Clef.Key) > 0 {
							nOptions.ClefKey = v.Nodes[i].Clef.Key
						}
						if len(v.Nodes[i].Clef.Password) > 0 {
							nOptions.ClefPassword = v.Nodes[i].Clef.Password
						}
						if len(v.Nodes[i].LibP2PKey) > 0 {
							nOptions.LibP2PKey = v.Nodes[i].LibP2PKey
						}
						if len(v.Nodes[i].SwarmKey) > 0 {
							nOptions.SwarmKey = v.Nodes[i].SwarmKey
						}

						errGroup.Go(func() error {
							return g.SetupNode(ctx, nName, nOptions, clusterConfig.Funding.Export())
						})
					}
				} else {
					for i := 0; i < v.Count; i++ {
						// set node name
						nName := fmt.Sprintf("%s-%d", ng, i)

						errGroup.Go(func() error {
							return g.SetupNode(ctx, nName, orchestration.NodeOptions{}, clusterConfig.Funding.Export())
						})
					}
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
				g, err := cluster.NodeGroup(ng)
				if err != nil {
					return nil, err
				}
				for i := 0; i < len(v.Nodes); i++ {
					// set node name
					nName := fmt.Sprintf("%s-%d", ng, i)
					if len(v.Nodes[i].Name) > 0 {
						nName = v.Nodes[i].Name
					}
					// set bootnodes
					beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
					if !ok {
						return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
					}
					bConfig := beeConfig.Export()
					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, clusterConfig.GetNamespace()) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					// set NodeOptions
					nOptions := orchestration.NodeOptions{
						Config: &bConfig,
					}
					if len(v.Nodes[i].Clef.Key) > 0 {
						nOptions.ClefKey = v.Nodes[i].Clef.Key
					}
					if len(v.Nodes[i].Clef.Password) > 0 {
						nOptions.ClefPassword = v.Nodes[i].Clef.Password
					}
					if len(v.Nodes[i].LibP2PKey) > 0 {
						nOptions.LibP2PKey = v.Nodes[i].LibP2PKey
					}
					if len(v.Nodes[i].SwarmKey) > 0 {
						nOptions.SwarmKey = v.Nodes[i].SwarmKey
					}

					if err := g.AddNode(nName, nOptions); err != nil {
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
				// set bootnodes
				beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
				if !ok {
					return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
				}
				bConfig := beeConfig.Export()
				bConfig.Bootnodes = bootnodes
				// add node group to the cluster
				gOptions := ngConfig.Export()
				gOptions.BeeConfig = &bConfig
				cluster.AddNodeGroup(ng, gOptions)

				// add nodes to the node group
				g, err := cluster.NodeGroup(ng)
				if err != nil {
					return nil, err
				}

				if len(v.Nodes) > 0 {
					for i := 0; i < len(v.Nodes); i++ {
						// set node name
						nName := fmt.Sprintf("%s-%d", ng, i)
						if len(v.Nodes[i].Name) > 0 {
							nName = v.Nodes[i].Name
						}
						// set NodeOptions
						nOptions := orchestration.NodeOptions{}
						if len(v.Nodes[i].Clef.Key) > 0 {
							nOptions.ClefKey = v.Nodes[i].Clef.Key
						}
						if len(v.Nodes[i].Clef.Password) > 0 {
							nOptions.ClefPassword = v.Nodes[i].Clef.Password
						}
						if len(v.Nodes[i].LibP2PKey) > 0 {
							nOptions.LibP2PKey = v.Nodes[i].LibP2PKey
						}
						if len(v.Nodes[i].SwarmKey) > 0 {
							nOptions.SwarmKey = v.Nodes[i].SwarmKey
						}

						if err := g.AddNode(nName, orchestration.NodeOptions{}); err != nil {
							return nil, fmt.Errorf("adding node %s: %w", nName, err)
						}
					}
				} else {
					for i := 0; i < v.Count; i++ {
						nName := fmt.Sprintf("%s-%d", ng, i)

						if err := g.AddNode(nName, orchestration.NodeOptions{}); err != nil {
							return nil, fmt.Errorf("adding node %s: %w", nName, err)
						}
					}
				}
			}
		}
	}

	return
}
