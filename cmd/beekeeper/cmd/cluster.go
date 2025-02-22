package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	orchestrationK8S "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
	"github.com/ethersphere/node-funder/pkg/funder"
)

const bootnodeMode string = "bootnode"

type nodeResult struct {
	ethAddress string
	err        error
}

func (c *command) deleteCluster(ctx context.Context, clusterName string, cfg *config.Config, deleteStorage bool) error {
	if clusterName == "" {
		return errMissingClusterName
	}

	clusterConfig, ok := cfg.Clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s not defined", clusterName)
	}

	cluster := orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterConfig.Export(), c.k8sClient, c.swapClient, c.log)

	// delete node groups
	for ngName, v := range clusterConfig.GetNodeGroups() {
		c.log.Infof("deleting %s node group", ngName)
		ngConfig, ok := cfg.NodeGroups[v.Config]
		if !ok {
			return fmt.Errorf("node group profile %s not defined", v.Config)
		}

		if v.Mode == bootnodeMode { // TODO: implement standalone mode
			// register node group
			cluster.AddNodeGroup(ngName, ngConfig.Export())

			// delete nodes from the node group
			ng, err := cluster.NodeGroup(ngName)
			if err != nil {
				return fmt.Errorf("get node group: %w", err)
			}

			for i := 0; i < len(v.Nodes); i++ {
				nName := fmt.Sprintf("%s-%d", ngName, i)
				if len(v.Nodes[i].Name) > 0 {
					nName = v.Nodes[i].Name
				}
				if err := ng.DeleteNode(ctx, nName); err != nil {
					return fmt.Errorf("deleting node %s from the node group %s: %w", nName, ngName, err)
				}

				if deleteStorage && *ngConfig.PersistenceEnabled {
					pvcName := fmt.Sprintf("data-%s-0", nName)
					if err := c.k8sClient.PVC.Delete(ctx, pvcName, clusterConfig.GetNamespace()); err != nil {
						return fmt.Errorf("deleting pvc %s: %w", pvcName, err)
					}
				}
			}
		} else {
			// register node group
			cluster.AddNodeGroup(ngName, ngConfig.Export())

			// delete nodes from the node group
			ng, err := cluster.NodeGroup(ngName)
			if err != nil {
				return err
			}

			if len(v.Nodes) > 0 {
				for i := 0; i < len(v.Nodes); i++ {
					nName := fmt.Sprintf("%s-%d", ngName, i)
					if len(v.Nodes[i].Name) > 0 {
						nName = v.Nodes[i].Name
					}

					if err := ng.DeleteNode(ctx, nName); err != nil {
						return fmt.Errorf("deleting node %s from the node group %s: %w", nName, ngName, err)
					}

					if deleteStorage && *ngConfig.PersistenceEnabled {
						pvcName := fmt.Sprintf("data-%s-0", nName)
						if err := c.k8sClient.PVC.Delete(ctx, pvcName, clusterConfig.GetNamespace()); err != nil {
							return fmt.Errorf("deleting pvc %s: %w", pvcName, err)
						}
					}
				}
			} else {
				for i := 0; i < v.Count; i++ {
					nName := fmt.Sprintf("%s-%d", ngName, i)
					if err := ng.DeleteNode(ctx, nName); err != nil {
						return fmt.Errorf("deleting node %s from the node group %s: %w", nName, ngName, err)
					}

					if deleteStorage && *ngConfig.PersistenceEnabled {
						pvcName := fmt.Sprintf("data-%s-0", nName)
						if err := c.k8sClient.PVC.Delete(ctx, pvcName, clusterConfig.GetNamespace()); err != nil {
							return fmt.Errorf("deleting pvc %s: %w", pvcName, err)
						}
					}
				}
			}
		}
	}

	return nil
}

func (c *command) setupCluster(ctx context.Context, clusterName string, startCluster bool) (cluster orchestration.Cluster, err error) {
	if clusterName == "" {
		return nil, errMissingClusterName
	}

	clusterConfig, ok := c.config.Clusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("cluster %s not defined", clusterName)
	}

	var chainNodeEndpoint string
	var walletKey string
	var fundOpts orchestration.FundingOptions

	if startCluster {
		if clusterConfig.IsUsingStaticEndpoints() {
			return nil, errors.New("static endpoints are not supported for starting the cluster")
		}

		if chainNodeEndpoint = c.globalConfig.GetString(optionNameGethURL); chainNodeEndpoint == "" {
			return nil, errBlockchainEndpointNotProvided
		}

		if walletKey = c.globalConfig.GetString(optionNameWalletKey); walletKey == "" {
			return nil, errors.New("wallet key not provided")
		}

		fundOpts = ensureFundingDefaults(clusterConfig.Funding.Export(), c.log)
	}

	cluster = orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterConfig.Export(), c.k8sClient, c.swapClient, c.log)

	nodeResultChan := make(chan nodeResult)
	defer close(nodeResultChan)

	inCluster := c.globalConfig.GetBool(optionNameInCluster)

	// setup bootnode node group
	fundAddresses, bootnodes, err := setupNodes(ctx, clusterConfig, c.config, true, cluster, startCluster, inCluster, "", nodeResultChan)
	if err != nil {
		return nil, fmt.Errorf("setup node group bootnode: %w", err)
	}

	// fund bootnode node group if cluster is started
	if startCluster {
		if err = fund(ctx, fundAddresses, chainNodeEndpoint, walletKey, fundOpts, c.log); err != nil {
			return nil, fmt.Errorf("funding node group bootnode: %w", err)
		}
		c.log.Infof("bootnode node group funded")
	}

	// setup other node groups
	fundAddresses, _, err = setupNodes(ctx, clusterConfig, c.config, false, cluster, startCluster, inCluster, bootnodes, nodeResultChan)
	if err != nil {
		return nil, fmt.Errorf("setup other node groups: %w", err)
	}

	// fund other node groups if cluster is started
	if startCluster {
		if err = fund(ctx, fundAddresses, chainNodeEndpoint, walletKey, fundOpts, c.log); err != nil {
			return nil, fmt.Errorf("fund other node groups: %w", err)
		}
		c.log.Infof("node groups funded")
	}

	c.log.WithField("use-static-endpoints", clusterConfig.IsUsingStaticEndpoints()).Infof("cluster %s setup completed", clusterName)

	return cluster, nil
}

func ensureFundingDefaults(fundOpts orchestration.FundingOptions, log logging.Logger) orchestration.FundingOptions {
	if fundOpts.Eth == 0 {
		fundOpts.Eth = 0.1 // default eth value
		log.Warningf("funding options, eth, is not provided, using default value %f", fundOpts.Eth)
	}
	if fundOpts.Bzz == 0 {
		fundOpts.Bzz = 100 // default bzz value
		log.Warningf("funding options, bzz, is not provided, using default value %f", fundOpts.Bzz)
	}
	log.Infof("fund options, eth: %f, bzz: %f", fundOpts.Eth, fundOpts.Bzz)
	return fundOpts
}

func setupNodes(ctx context.Context,
	clusterConfig config.Cluster,
	cfg *config.Config,
	bootnode bool,
	cluster orchestration.Cluster,
	startCluster bool,
	inCluster bool,
	bootnodesIn string,
	nodeResultCh chan nodeResult,
) (fundAddresses []string, bootnodesOut string, err error) {
	var nodeCount uint32

	for ngName, v := range clusterConfig.GetNodeGroups() {

		if (v.Mode != bootnodeMode && bootnode) || (v.Mode == bootnodeMode && !bootnode) {
			continue
		}

		ngConfig, ok := cfg.NodeGroups[v.Config]
		if !ok {
			return nil, "", fmt.Errorf("node group profile %s not defined", v.Config)
		}
		ngOptions := ngConfig.Export()

		beeConfig, ok := cfg.BeeConfigs[v.BeeConfig]
		if !ok {
			return nil, "", fmt.Errorf("bee profile %s not defined", v.BeeConfig)
		}
		bConfig := beeConfig.Export()

		if !bootnode {
			bConfig.Bootnodes = bootnodesIn
			ngOptions.BeeConfig = &bConfig
		}

		cluster.AddNodeGroup(ngName, ngOptions)

		// start nodes in the node group
		ng, err := cluster.NodeGroup(ngName)
		if err != nil {
			return nil, "", fmt.Errorf("get node group: %w", err)
		}

		if clusterConfig.IsUsingStaticEndpoints() {
			for nodeName, endpoint := range v.GetEndpoints() {
				beeOpt := orchestration.WithURL(endpoint.APIURL)
				nodeCount++
				go setupOrAddNode(ctx, false, inCluster, ng, nodeName, orchestration.NodeOptions{
					Config: &bConfig,
				}, nodeResultCh, beeOpt)
			}
			continue
		}

		for i, node := range v.Nodes {
			// set node name
			nodeName := fmt.Sprintf("%s-%d", ngName, i)
			if len(node.Name) > 0 {
				nodeName = node.Name
			}

			var nodeOpts orchestration.NodeOptions
			if bootnode {
				// set bootnodes
				bConfig.Bootnodes = fmt.Sprintf(node.Bootnodes, clusterConfig.GetNamespace()) // TODO: improve bootnode management, support more than 2 bootnodes
				bootnodesOut += bootnodesIn + bConfig.Bootnodes + " "
				nodeOpts = setupNodeOptions(node, &bConfig)
			} else {
				nodeOpts = setupNodeOptions(node, nil)
			}
			nodeCount++
			go setupOrAddNode(ctx, startCluster, inCluster, ng, nodeName, nodeOpts, nodeResultCh, orchestration.WithNoOptions())
		}

		if len(v.Nodes) == 0 && !bootnode {
			for i := 0; i < v.Count; i++ {
				// set node name
				nodeName := fmt.Sprintf("%s-%d", ngName, i)
				nodeCount++
				go setupOrAddNode(ctx, startCluster, inCluster, ng, nodeName, orchestration.NodeOptions{}, nodeResultCh, orchestration.WithNoOptions())
			}
		}
	}

	// wait for nodes to be setup and get their eth addresses
	// or wait for nodes to be added and check for errors
	for i := uint32(0); i < nodeCount; i++ {
		nodeResult := <-nodeResultCh
		if nodeResult.err != nil {
			return nil, "", fmt.Errorf("setup or add node result: %w", nodeResult.err)
		}
		if nodeResult.ethAddress != "" {
			fundAddresses = append(fundAddresses, nodeResult.ethAddress)
		}
	}

	return fundAddresses, bootnodesOut, nil
}

func setupOrAddNode(ctx context.Context,
	startCluster bool,
	inCluster bool,
	ng orchestration.NodeGroup,
	nodeName string,
	nodeOpts orchestration.NodeOptions,
	ch chan<- nodeResult,
	beeOpt orchestration.BeeClientOption,
) {
	if startCluster {
		ethAddress, err := ng.DeployNode(ctx, nodeName, inCluster, nodeOpts)
		ch <- nodeResult{
			ethAddress: ethAddress,
			err:        err,
		}
	} else {
		ch <- nodeResult{
			err: ng.AddNode(ctx, nodeName, inCluster, nodeOpts, beeOpt),
		}
	}
}

func setupNodeOptions(node config.ClusterNode, bConfig *orchestration.Config) orchestration.NodeOptions {
	return orchestration.NodeOptions{
		Config:    bConfig,
		LibP2PKey: node.LibP2PKey,
		SwarmKey:  node.SwarmKey,
	}
}

func fund(
	ctx context.Context,
	fundAddresses []string,
	chainNodeEndpoint string,
	walletKey string,
	fundOpts orchestration.FundingOptions,
	log logging.Logger,
) error {
	return funder.Fund(ctx, funder.Config{
		Addresses:         fundAddresses,
		ChainNodeEndpoint: chainNodeEndpoint,
		WalletKey:         walletKey,
		MinAmounts: funder.MinAmounts{
			NativeCoin: fundOpts.Eth,
			SwarmToken: fundOpts.Bzz,
		},
	}, nil, nil, funder.WithLoggerOption(log))
}
