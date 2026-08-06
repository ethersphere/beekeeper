package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	orchestrationK8S "github.com/ethersphere/beekeeper/pkg/orchestration/k8s"
	"github.com/spf13/cobra"
)

func (c *command) initScaleCmd() (err error) {
	const (
		optionNameNodeGroup = "node-group"
		optionNameCount     = "count"
		optionNameWalletKey = "wallet-key"
		optionNameTimeout   = "timeout"
	)

	cmd := &cobra.Command{
		Use:   "scale",
		Short: "scales up a node group in a running Bee cluster",
		Long: `Scales up a node group in a running Bee cluster by deploying additional nodes.

Existing nodes are left untouched: for each index in [0, count), the command checks
whether a node with that index already exists in the node group and only deploys the
ones that are missing. This makes it safe to re-run and safe against a partially
completed previous scale.

Only scaling up (increasing count) is supported; --count must be greater than the
node group's currently deployed size.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx, cancel := context.WithTimeout(cmd.Context(), c.globalConfig.GetDuration(optionNameTimeout))
			defer cancel()

			clusterName := c.globalConfig.GetString(optionNameClusterName)
			ngName := c.globalConfig.GetString(optionNameNodeGroup)
			count := c.globalConfig.GetInt(optionNameCount)

			if ngName == "" {
				return fmt.Errorf("node group name not provided")
			}
			if count <= 0 {
				return fmt.Errorf("count must be greater than 0")
			}

			chainNodeEndpoint := c.globalConfig.GetString(optionNameGethURL)
			if chainNodeEndpoint == "" {
				return errBlockchainEndpointNotProvided
			}

			walletKey := c.globalConfig.GetString(optionNameWalletKey)
			if walletKey == "" {
				return fmt.Errorf("wallet key not provided")
			}

			start := time.Now()
			err = c.scaleNodeGroup(ctx, clusterName, ngName, count, chainNodeEndpoint, walletKey)
			c.log.Infof("scale took %s", time.Since(start))
			return err
		},
		PreRunE: c.preRunE,
	}

	cmd.Flags().String(optionNameClusterName, "", "cluster name")
	cmd.Flags().String(optionNameNodeGroup, "", "node group to scale up. Required")
	cmd.Flags().Int(optionNameCount, 0, "new target node count for the node group. Required")
	cmd.Flags().String(optionNameWalletKey, "", "Hex-encoded private key for the Bee node wallet. Required.")
	cmd.Flags().Duration(optionNameTimeout, 30*time.Minute, "timeout")

	c.root.AddCommand(cmd)

	return nil
}

// scaleNodeGroup deploys the nodes missing between the node group's current size
// and the requested count, leaving already-deployed nodes untouched.
func (c *command) scaleNodeGroup(ctx context.Context, clusterName, ngName string, count int, chainNodeEndpoint, walletKey string) error {
	if clusterName == "" {
		return errMissingClusterName
	}

	clusterConfig, ok := c.config.Clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s not defined", clusterName)
	}

	if clusterConfig.IsUsingStaticEndpoints() {
		return fmt.Errorf("static endpoints are not supported for scaling")
	}

	ngv, ok := clusterConfig.GetNodeGroups()[ngName]
	if !ok {
		return fmt.Errorf("node group %s not defined in cluster %s", ngName, clusterName)
	}
	if ngv.Mode == bootnodeMode {
		return fmt.Errorf("scaling bootnode node groups is not supported")
	}
	if len(ngv.Nodes) > 0 {
		return fmt.Errorf("node group %s uses explicit nodes, not count-based scaling", ngName)
	}

	ngConfig, ok := c.config.NodeGroups[ngv.Config]
	if !ok {
		return fmt.Errorf("node group profile %s not defined", ngv.Config)
	}

	beeConfig, ok := c.config.BeeConfigs[ngv.BeeConfig]
	if !ok {
		return fmt.Errorf("bee profile %s not defined", ngv.BeeConfig)
	}
	bConfig := beeConfig.Export()

	if bConfig.Bootnodes == nil || len(*bConfig.Bootnodes) == 0 {
		bootnodes, err := bootnodesForCluster(clusterConfig, c.config)
		if err != nil {
			return fmt.Errorf("resolve bootnodes: %w", err)
		}
		if bootnodes != "" {
			bConfig.Bootnodes = &[]string{bootnodes}
		}
	}

	ngOptions := ngConfig.Export()
	ngOptions.BeeConfig = &bConfig

	namespace := clusterConfig.GetNamespace()

	newNames := make([]string, 0, count)
	for i := range count {
		name := fmt.Sprintf("%s-%d", ngName, i)

		exists, err := c.k8sClient.StatefulSet.Exists(ctx, name, namespace)
		if err != nil {
			return fmt.Errorf("checking node %s: %w", name, err)
		}
		if exists {
			continue
		}

		newNames = append(newNames, name)
	}

	if len(newNames) == 0 {
		c.log.Infof("node group %s already has %d or more nodes, nothing to do", ngName, count)
		return nil
	}

	c.log.Infof("scaling node group %s: deploying %d new node(s): %v", ngName, len(newNames), newNames)

	cluster := orchestrationK8S.NewCluster(clusterConfig.GetName(), clusterConfig.Export(), c.k8sClient, c.swapClient, c.log)
	cluster.AddNodeGroup(ngName, ngOptions)

	ng, err := cluster.NodeGroup(ngName)
	if err != nil {
		return fmt.Errorf("get node group: %w", err)
	}

	inCluster := c.globalConfig.GetBool(optionNameInCluster)

	type result struct {
		ethAddress string
		err        error
	}
	resultChan := make(chan result)
	defer close(resultChan)

	for _, name := range newNames {
		go func(name string) {
			ethAddress, err := ng.DeployNode(ctx, name, inCluster, orchestration.NodeOptions{})
			resultChan <- result{ethAddress: ethAddress, err: err}
		}(name)
	}

	var fundAddresses []string
	for range newNames {
		r := <-resultChan
		if r.err != nil {
			return fmt.Errorf("deploy node: %w", r.err)
		}
		if r.ethAddress != "" {
			fundAddresses = append(fundAddresses, r.ethAddress)
		}
	}

	if len(fundAddresses) > 0 {
		fundOpts := ensureFundingDefaults(clusterConfig.Funding.Export(), c.log)
		if err := fund(ctx, fundAddresses, chainNodeEndpoint, walletKey, fundOpts, c.log); err != nil {
			return fmt.Errorf("funding new nodes: %w", err)
		}
		c.log.Infof("new nodes funded")
	}

	c.log.Infof("node group %s scaled to %d nodes", ngName, count)

	return nil
}

// bootnodesForCluster resolves the bootnode multiaddr for a cluster the same way
// cluster setup does, without deploying or contacting the bootnode node group.
func bootnodesForCluster(clusterConfig config.Cluster, cfg *config.Config) (string, error) {
	for _, v := range clusterConfig.GetNodeGroups() {
		if v.Mode != bootnodeMode {
			continue
		}
		for _, node := range v.Nodes {
			if node.Bootnodes == "" {
				continue
			}
			return fmt.Sprintf(node.Bootnodes, clusterConfig.GetNamespace()), nil
		}
	}
	return "", nil
}
