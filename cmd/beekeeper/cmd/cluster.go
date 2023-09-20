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

	cluster := orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterOptions, c.log)

	// delete node groups
	for ng, v := range clusterConfig.GetNodeGroups() {
		c.log.Infof("deleting %s node group", ng)
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

	cluster = orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterOptions, c.log)
	bootnodes := ""

	errGroup := new(errgroup.Group)

	for ng, v := range clusterConfig.GetNodeGroups() {
		ngConfig, ok := cfg.NodeGroups[v.Config]
		if !ok {
			return nil, fmt.Errorf("node group profile %s not defined", v.Config)
		}

		if v.Mode == "bootnode" { // TODO: implement standalone mode
			beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
			if !ok {
				return nil, fmt.Errorf("bee profile %s not defined", v.BeeConfig)
			}

			// add node group to the cluster
			cluster.AddNodeGroup(ng, ngConfig.Export())

			// start nodes in the node group
			g, err := cluster.NodeGroup(ng)
			if err != nil {
				return nil, err
			}

			for i, node := range v.Nodes {
				// set node name
				nName := fmt.Sprintf("%s-%d", ng, i)
				if len(node.Name) > 0 {
					nName = node.Name
				}

				// set bootnodes
				bConfig := beeConfig.Export()
				bConfig.Bootnodes = fmt.Sprintf(node.Bootnodes, clusterConfig.GetNamespace()) // TODO: improve bootnode management, support more than 2 bootnodes
				bootnodes += bConfig.Bootnodes + " "

				// set NodeOptions
				nOptions := orchestration.NodeOptions{
					Config: &bConfig,
				}
				if len(node.Clef.Key) > 0 {
					nOptions.ClefKey = node.Clef.Key
				}
				if len(node.Clef.Password) > 0 {
					nOptions.ClefPassword = node.Clef.Password
				}
				if len(node.LibP2PKey) > 0 {
					nOptions.LibP2PKey = node.LibP2PKey
				}
				if len(node.SwarmKey) > 0 {
					nOptions.SwarmKey = orchestration.EncryptedKey(node.SwarmKey)
				}

				errGroup.Go(func() error {
					if start {
						return g.SetupNode(ctx, nName, nOptions, clusterConfig.Funding.Export())
					} else {
						return g.AddNode(ctx, nName, nOptions)
					}
				})
			}
		}
	}

	if err := errGroup.Wait(); err != nil {
		return nil, fmt.Errorf("starting node group bootnode: %w", err)
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

			for i, node := range v.Nodes {
				// set node name
				nName := fmt.Sprintf("%s-%d", ng, i)
				if len(node.Name) > 0 {
					nName = node.Name
				}
				// set NodeOptions
				nOptions := orchestration.NodeOptions{}
				if len(node.Clef.Key) > 0 {
					nOptions.ClefKey = node.Clef.Key
				}
				if len(node.Clef.Password) > 0 {
					nOptions.ClefPassword = node.Clef.Password
				}
				if len(node.LibP2PKey) > 0 {
					nOptions.LibP2PKey = node.LibP2PKey
				}
				if len(node.SwarmKey) > 0 {
					nOptions.SwarmKey = orchestration.EncryptedKey(node.SwarmKey)
				}

				errGroup.Go(func() error {
					if start {
						return g.SetupNode(ctx, nName, nOptions, clusterConfig.Funding.Export())
					} else {
						return g.AddNode(ctx, nName, nOptions)
					}
				})
			}

			if len(v.Nodes) == 0 {
				for i := 0; i < v.Count; i++ {
					// set node name
					nName := fmt.Sprintf("%s-%d", ng, i)

					errGroup.Go(func() error {
						if start {
							return g.SetupNode(ctx, nName, orchestration.NodeOptions{}, clusterConfig.Funding.Export())
						} else {
							return g.AddNode(ctx, nName, orchestration.NodeOptions{})
						}
					})
				}
			}
		}
	}

	if err := errGroup.Wait(); err != nil {
		return nil, fmt.Errorf("starting node groups: %w", err)
	}

	return
}
