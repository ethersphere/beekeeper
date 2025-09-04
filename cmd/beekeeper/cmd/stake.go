package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/node"
	"github.com/ethersphere/beekeeper/pkg/orchestration"
	"github.com/spf13/cobra"
)

const (
	optionNameAmount     = "amount"
	optionNameParallel   = "parallel"
	optionNameNodeGroups = "node-groups"
	maxParallel          = 10
)

var (
	errMissingStakeAmount = errors.New("stake amount not provided")
	errInvalidAmount      = errors.New("invalid stake amount")
)

func (c *command) initStakeCmd() (err error) {
	cmd := &cobra.Command{
		Use:   "stake",
		Short: "Stakes Bee nodes",
		Long:  `Stakes Bee nodes with BZZ tokens and ETH for Bee node operations.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return cmd.Help()
		},
		PreRunE: c.preRunE,
	}

	cmd.AddCommand(c.initStakeDeposit())
	cmd.AddCommand(c.initStakeGet())
	cmd.AddCommand(c.initStakeWithdraw())

	c.root.AddCommand(cmd)

	return nil
}

func (c *command) initStakeDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "Deposit stake on Bee nodes",
		Long:  `Deposits a specified amount of BZZ as stake on targeted Bee nodes.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			amount, err := cmd.Flags().GetString(optionNameAmount)
			if err != nil {
				return fmt.Errorf("error reading amount flag: %w", err)
			}

			if amount == "" {
				return errMissingStakeAmount
			}

			stakeAmount, ok := new(big.Int).SetString(amount, 10)
			if !ok {
				return errInvalidAmount
			}

			if stakeAmount.Cmp(big.NewInt(0)) <= 0 {
				return errInvalidAmount
			}

			namespace, err := cmd.Flags().GetString(optionNameNamespace)
			if err != nil {
				return fmt.Errorf("error reading namespace flag: %w", err)
			}

			clusterName, err := cmd.Flags().GetString(optionNameClusterName)
			if err != nil {
				return fmt.Errorf("error reading cluster-name flag: %w", err)
			}

			if clusterName == "" && namespace == "" {
				return fmt.Errorf("either cluster-name or namespace must be provided")
			}

			ctx := context.Background()
			var clients map[string]*bee.Client
			var nodes node.NodeList

			if namespace != "" {
				fmt.Printf("Targeting namespace: %s\n", namespace)

				labelSelector, err := cmd.Flags().GetString(optionNameLabelSelector)
				if err != nil {
					return fmt.Errorf("error reading label-selector flag: %w", err)
				}

				nodeClient := node.New(&node.ClientConfig{
					Log:            c.log,
					HTTPClient:     c.httpClient,
					K8sClient:      c.k8sClient,
					BeeClients:     nil,
					Namespace:      namespace,
					LabelSelector:  labelSelector,
					DeploymentType: node.DeploymentTypeBeekeeper,
					InCluster:      c.globalConfig.GetBool(optionNameInCluster),
					UseNamespace:   true,
				})

				nodes, err = nodeClient.GetNodes(ctx)
				if err != nil {
					return fmt.Errorf("getting nodes: %w", err)
				}

				clients = make(map[string]*bee.Client)
			} else {
				fmt.Printf("Targeting cluster: %s\n", clusterName)
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("failed to setup cluster %s: %w", clusterName, err)
				}

				allClients, err := cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to get node clients: %w", err)
				}

				nodeGroups, err := cmd.Flags().GetStringSlice(optionNameNodeGroups)
				if err != nil {
					return fmt.Errorf("error reading node-groups flag: %w", err)
				}

				if len(nodeGroups) > 0 {
					fmt.Printf("Filtering by node groups: %v\n", nodeGroups)
					clients = c.filterClientsByNodeGroups(cluster, allClients, nodeGroups)
				} else {
					fmt.Printf("No node groups specified, defaulting to 'bee' nodes for staking\n")
					clients = c.filterClientsByNodeGroups(cluster, allClients, []string{"bee"})
				}
			}

			parallel, err := cmd.Flags().GetInt(optionNameParallel)
			if err != nil {
				fmt.Printf("Warning: Could not read parallel flag, using default value of 5\n")
				parallel = 5
			}

			if parallel <= 0 {
				fmt.Printf("Warning: Invalid parallel value (%d), using default value of 5\n", parallel)
				parallel = 5
			}

			nodeCount := len(clients)
			if namespace != "" {
				nodeCount = len(nodes)
			}

			if parallel > nodeCount {
				fmt.Printf("Info: Parallel value (%d) is greater than number of nodes (%d), using %d\n", parallel, nodeCount, nodeCount)
				parallel = nodeCount
			}

			if parallel > maxParallel {
				fmt.Printf("Info: Parallel value (%d) is too high, capping at %d to prevent network overload\n", parallel, maxParallel)
				parallel = maxParallel
			}

			fmt.Printf("Starting stake deposit of %s WEI on %d nodes with %d parallel operations...\n", amount, nodeCount, parallel)

			var errorCount int
			var mu sync.Mutex
			semaphore := make(chan struct{}, parallel)
			var wg sync.WaitGroup

			if namespace != "" {
				for _, n := range nodes {
					wg.Add(1)
					go func(node node.Node) {
						defer wg.Done()
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						fmt.Printf("Depositing stake on node %s...\n", node.Name())

						txHash, err := node.Client().Stake.DepositStake(ctx, stakeAmount)
						if err != nil {
							mu.Lock()
							errorCount++
							mu.Unlock()
							fmt.Printf("%s\n", c.formatStakeError(node.Name(), err))
							return
						}

						fmt.Printf("Successfully deposited stake on node %s, transaction: %s\n", node.Name(), txHash)
					}(n)
				}
			} else {
				for nodeName, client := range clients {
					wg.Add(1)
					go func(name string, cl *bee.Client) {
						defer wg.Done()
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						fmt.Printf("Depositing stake on node %s...\n", name)

						txHash, err := cl.DepositStake(ctx, stakeAmount)
						if err != nil {
							mu.Lock()
							errorCount++
							mu.Unlock()
							fmt.Printf("%s\n", c.formatStakeError(name, err))
							return
						}

						fmt.Printf("Successfully deposited stake on node %s, transaction: %s\n", name, txHash)
					}(nodeName, client)
				}
			}

			wg.Wait()

			if errorCount > 0 {
				return fmt.Errorf("stake deposit completed with %d errors", errorCount)
			}

			fmt.Printf("Stake deposit completed successfully on all %d nodes!\n", nodeCount)
			return nil
		},
	}

	cmd.Flags().String(optionNameAmount, "", "Stake amount in WEI (required)")
	if err := cmd.MarkFlagRequired(optionNameAmount); err != nil {
		return nil
	}
	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name)")
	cmd.Flags().String(optionNameLabelSelector, "app.kubernetes.io/name=bee", "Kubernetes label selector for filtering resources")
	cmd.Flags().StringSlice(optionNameNodeGroups, nil, "List of node groups to target (applies to all groups if not set)")
	cmd.Flags().Int(optionNameParallel, 5, "Number of parallel operations (default: 5, max: number of nodes)")

	return cmd
}

// formatStakeError formats stake-related errors with user-friendly messages
func (c *command) formatStakeError(nodeName string, err error) string {
	errorStr := err.Error()

	if strings.Contains(errorStr, "out of funds") {
		return fmt.Sprintf("node %s: insufficient BZZ balance (fund the node wallet first)", nodeName)
	} else if strings.Contains(errorStr, "insufficient stake amount") {
		return fmt.Sprintf("node %s: stake amount too low (increase the amount)", nodeName)
	} else if strings.Contains(errorStr, "503") {
		return fmt.Sprintf("node %s: service temporarily unavailable (node might be starting up)", nodeName)
	} else {
		return fmt.Sprintf("node %s: %v", nodeName, err)
	}
}

func (c *command) initStakeGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get current stake amounts from Bee nodes",
		Long:  "Retrieves the current stake amounts from targeted Bee nodes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := cmd.Flags().GetString(optionNameNamespace)
			if err != nil {
				return fmt.Errorf("error reading namespace flag: %w", err)
			}

			clusterName, err := cmd.Flags().GetString(optionNameClusterName)
			if err != nil {
				return fmt.Errorf("error reading cluster-name flag: %w", err)
			}

			if clusterName == "" && namespace == "" {
				return fmt.Errorf("either cluster-name or namespace must be provided")
			}

			ctx := context.Background()
			var clients map[string]*bee.Client
			var nodes node.NodeList

			if namespace != "" {
				fmt.Printf("Targeting namespace: %s\n", namespace)

				labelSelector, err := cmd.Flags().GetString(optionNameLabelSelector)
				if err != nil {
					return fmt.Errorf("error reading label-selector flag: %w", err)
				}

				nodeClient := node.New(&node.ClientConfig{
					Log:            c.log,
					HTTPClient:     c.httpClient,
					K8sClient:      c.k8sClient,
					BeeClients:     nil,
					Namespace:      namespace,
					LabelSelector:  labelSelector,
					DeploymentType: node.DeploymentTypeBeekeeper,
					InCluster:      c.globalConfig.GetBool(optionNameInCluster),
					UseNamespace:   true,
				})

				nodes, err = nodeClient.GetNodes(ctx)
				if err != nil {
					return fmt.Errorf("getting nodes: %w", err)
				}

				clients = make(map[string]*bee.Client)
			} else {
				fmt.Printf("Targeting cluster: %s\n", clusterName)
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("failed to setup cluster %s: %w", clusterName, err)
				}

				allClients, err := cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to get node clients: %w", err)
				}

				nodeGroups, err := cmd.Flags().GetStringSlice(optionNameNodeGroups)
				if err != nil {
					return fmt.Errorf("error reading node-groups flag: %w", err)
				}

				if len(nodeGroups) > 0 {
					fmt.Printf("Filtering by node groups: %v\n", nodeGroups)
					clients = c.filterClientsByNodeGroups(cluster, allClients, nodeGroups)
				} else {
					fmt.Printf("No node groups specified, defaulting to 'bee' nodes for staking\n")
					clients = c.filterClientsByNodeGroups(cluster, allClients, []string{"bee"})
				}
			}

			nodeCount := len(clients)
			if namespace != "" {
				nodeCount = len(nodes)
			}
			fmt.Printf("Found %d nodes\n", nodeCount)

			parallel, err := cmd.Flags().GetInt(optionNameParallel)
			if err != nil {
				fmt.Printf("Warning: Could not read parallel flag, using default value of 5\n")
				parallel = 5
			}

			if parallel <= 0 {
				fmt.Printf("Warning: Invalid parallel value (%d), using default value of 5\n", parallel)
				parallel = 5
			}

			if parallel > nodeCount {
				fmt.Printf("Info: Parallel value (%d) is greater than number of nodes (%d), using %d\n", parallel, nodeCount, nodeCount)
				parallel = nodeCount
			}

			if parallel > maxParallel {
				fmt.Printf("Info: Parallel value (%d) is too high, capping at %d to prevent network overload\n", parallel, maxParallel)
				parallel = maxParallel
			}

			fmt.Printf("Getting stake amounts from %d nodes with %d parallel operations...\n", nodeCount, parallel)

			var errorCount int
			var mu sync.Mutex
			semaphore := make(chan struct{}, parallel)
			var wg sync.WaitGroup

			if namespace != "" {
				for _, n := range nodes {
					wg.Add(1)
					go func(node node.Node) {
						defer wg.Done()
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						fmt.Printf("Getting stake from node %s...\n", node.Name())

						stakeAmount, err := node.Client().Stake.GetStakedAmount(ctx)
						if err != nil {
							mu.Lock()
							errorCount++
							mu.Unlock()
							fmt.Printf("Error getting stake from node %s: %v\n", node.Name(), err)
							return
						}

						fmt.Printf("Node %s: %s WEI staked\n", node.Name(), stakeAmount.String())
					}(n)
				}
			} else {
				for nodeName, client := range clients {
					wg.Add(1)
					go func(name string, cl *bee.Client) {
						defer wg.Done()
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						fmt.Printf("Getting stake from node %s...\n", name)

						stakeAmount, err := cl.GetStake(ctx)
						if err != nil {
							mu.Lock()
							errorCount++
							mu.Unlock()
							fmt.Printf("Error getting stake from node %s: %v\n", name, err)
							return
						}

						fmt.Printf("Node %s: %s WEI staked\n", name, stakeAmount.String())
					}(nodeName, client)
				}
			}

			wg.Wait()

			if errorCount > 0 {
				return fmt.Errorf("stake retrieval completed with %d errors", errorCount)
			}

			fmt.Printf("Stake retrieval completed successfully from all %d nodes!\n", nodeCount)
			return nil
		},
	}

	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name)")
	cmd.Flags().String(optionNameLabelSelector, "app.kubernetes.io/name=bee", "Kubernetes label selector for filtering resources")
	cmd.Flags().StringSlice(optionNameNodeGroups, nil, "List of node groups to target (applies to all groups if not set)")
	cmd.Flags().Int(optionNameParallel, 5, "Number of parallel operations (default: 5, max: number of nodes)")

	return cmd
}

func (c *command) initStakeWithdraw() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "withdraw stake from Bee nodes",
		Long:  "Withdraws (migrates) stake from targeted Bee nodes. This operation migrates the stake to the node's wallet.",
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := cmd.Flags().GetString(optionNameNamespace)
			if err != nil {
				return fmt.Errorf("error reading namespace flag: %w", err)
			}

			clusterName, err := cmd.Flags().GetString(optionNameClusterName)
			if err != nil {
				return fmt.Errorf("error reading cluster-name flag: %w", err)
			}

			if clusterName == "" && namespace == "" {
				return fmt.Errorf("either cluster-name or namespace must be provided")
			}

			ctx := context.Background()
			var clients map[string]*bee.Client
			var nodes node.NodeList

			if namespace != "" {
				fmt.Printf("Targeting namespace: %s\n", namespace)

				labelSelector, err := cmd.Flags().GetString(optionNameLabelSelector)
				if err != nil {
					return fmt.Errorf("error reading label-selector flag: %w", err)
				}

				nodeClient := node.New(&node.ClientConfig{
					Log:            c.log,
					HTTPClient:     c.httpClient,
					K8sClient:      c.k8sClient,
					BeeClients:     nil,
					Namespace:      namespace,
					LabelSelector:  labelSelector,
					DeploymentType: node.DeploymentTypeBeekeeper,
					InCluster:      c.globalConfig.GetBool(optionNameInCluster),
					UseNamespace:   true,
				})

				nodes, err = nodeClient.GetNodes(ctx)
				if err != nil {
					return fmt.Errorf("getting nodes: %w", err)
				}

				clients = make(map[string]*bee.Client)
			} else {
				fmt.Printf("Targeting cluster: %s\n", clusterName)
				cluster, err := c.setupCluster(ctx, clusterName, false)
				if err != nil {
					return fmt.Errorf("failed to setup cluster %s: %w", clusterName, err)
				}

				allClients, err := cluster.NodesClients(ctx)
				if err != nil {
					return fmt.Errorf("failed to get node clients: %w", err)
				}

				nodeGroups, err := cmd.Flags().GetStringSlice(optionNameNodeGroups)
				if err != nil {
					return fmt.Errorf("error reading node-groups flag: %w", err)
				}

				if len(nodeGroups) > 0 {
					fmt.Printf("Filtering by node groups: %v\n", nodeGroups)
					clients = c.filterClientsByNodeGroups(cluster, allClients, nodeGroups)
				} else {
					fmt.Printf("No node groups specified, defaulting to 'bee' nodes for staking\n")
					clients = c.filterClientsByNodeGroups(cluster, allClients, []string{"bee"})
				}
			}

			nodeCount := len(clients)
			if namespace != "" {
				nodeCount = len(nodes)
			}
			fmt.Printf("Found %d nodes\n", nodeCount)

			parallel, err := cmd.Flags().GetInt(optionNameParallel)
			if err != nil {
				fmt.Printf("Warning: Could not read parallel flag, using default value of 5\n")
				parallel = 5
			}

			if parallel <= 0 {
				fmt.Printf("Warning: Invalid parallel value (%d), using default value of 5\n", parallel)
				parallel = 5
			}

			if parallel > nodeCount {
				fmt.Printf("Info: Parallel value (%d) is greater than number of nodes (%d), using %d\n", parallel, nodeCount, nodeCount)
				parallel = nodeCount
			}

			if parallel > maxParallel {
				fmt.Printf("Info: Parallel value (%d) is too high, capping at %d to prevent network overload\n", parallel, maxParallel)
				parallel = maxParallel
			}

			fmt.Printf("Starting stake withdrawal from %d nodes with %d parallel operations...\n", nodeCount, parallel)

			var errorCount int
			var mu sync.Mutex
			semaphore := make(chan struct{}, parallel)
			var wg sync.WaitGroup

			if namespace != "" {
				for _, n := range nodes {
					wg.Add(1)
					go func(node node.Node) {
						defer wg.Done()
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						fmt.Printf("Withdrawing stake from node %s...\n", node.Name())

						txHash, err := node.Client().Stake.MigrateStake(ctx)
						if err != nil {
							mu.Lock()
							errorCount++
							mu.Unlock()
							fmt.Printf("Error withdrawing stake from node %s: %v\n", node.Name(), err)
							return
						}

						fmt.Printf("Successfully withdrew stake from node %s, transaction: %s\n", node.Name(), txHash)
					}(n)
				}
			} else {
				for nodeName, client := range clients {
					wg.Add(1)
					go func(name string, cl *bee.Client) {
						defer wg.Done()
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						fmt.Printf("Withdrawing stake from node %s...\n", name)

						txHash, err := cl.MigrateStake(ctx)
						if err != nil {
							mu.Lock()
							errorCount++
							mu.Unlock()
							fmt.Printf("Error withdrawing stake from node %s: %v\n", name, err)
							return
						}

						fmt.Printf("Successfully withdrew stake from node %s, transaction: %s\n", name, txHash)
					}(nodeName, client)
				}
			}

			wg.Wait()

			if errorCount > 0 {
				return fmt.Errorf("stake withdrawal completed with %d errors", errorCount)
			}

			fmt.Printf("Stake withdrawal completed successfully from all %d nodes!\n", nodeCount)
			return nil
		},
	}

	cmd.Flags().String(optionNameClusterName, "", "Target Beekeeper cluster name")
	cmd.Flags().StringP(optionNameNamespace, "n", "", "Kubernetes namespace (overrides cluster name)")
	cmd.Flags().String(optionNameLabelSelector, "app.kubernetes.io/name=bee", "Kubernetes label selector for filtering resources")
	cmd.Flags().StringSlice(optionNameNodeGroups, nil, "List of node groups to target (applies to all groups if not set)")
	cmd.Flags().Int(optionNameParallel, 5, "Number of parallel operations (default: 5, max: number of nodes)")

	return cmd
}

func (c *command) filterClientsByNodeGroups(cluster orchestration.Cluster, allClients map[string]*bee.Client, nodeGroups []string) map[string]*bee.Client {
	nodeGroupsMap := cluster.NodeGroups()
	var targetNodes []string

	for _, nodeGroup := range nodeGroups {
		group, ok := nodeGroupsMap[nodeGroup]
		if !ok {
			c.log.Debugf("node group %s not found in cluster", nodeGroup)
			continue
		}
		targetNodes = append(targetNodes, group.NodesSorted()...)
	}

	// Filter clients to only include nodes from specified groups
	filteredClients := make(map[string]*bee.Client)
	for _, nodeName := range targetNodes {
		if client, exists := allClients[nodeName]; exists {
			filteredClients[nodeName] = client
		}
	}

	return filteredClients
}
