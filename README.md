# Ethereum Swarm Beekeeper

[![Go](https://github.com/ethersphere/beekeeper/workflows/Go/badge.svg)](https://github.com/ethersphere/beekeeper/actions)

## Table of Contents

- [Introduction](#introduction)
- [Requirements](#requirements)
- [Installation](#installation)
- [Run unit tests](#run-unit-tests)
- [Configuration](#configuration)
- [Config file](#config-file)
- [Config directory](#config-directory)
  - [Inheritance](#inheritance)
  - [Action types](#action-types)
- [Usage](#usage)
  - [check](#check)
  - [create](#create)
  - [delete](#delete)
  - [fund](#fund)
  - [print](#print)
  - [simulate](#simulate)
  - [version](#version)
  - [node-funder](#node-funder)
  - [node-operator](#node-operator)
  - [restart](#restart)
  - [stamper](#stamper)
- [Global flags](#global-flags)
- [Public Testnet Checks](#public-testnet-checks)
  - [One by one](#one-by-one)
  - [All at once, sequentially](#all-at-once-sequentially)

## Introduction

**Ethereum Swarm Beekeeper** is an orchestration and testing tool for managing [Ethereum Swarm Bee](https://github.com/ethersphere/bee) clusters. It enables:

- **Cluster Management**: Automate creation and deletion of Bee clusters in Kubernetes.
- **Integration Checks**: Run tests (e.g., `pingpong`, `pushsync`) to validate network behavior.
- **Static Endpoints Support**: Execute checks without Kubernetes by using static Bee node endpoints.
- **Node Funding**: Automate funding Bee nodes with ETH and BZZ tokens (Kubernetes optional).
- **Dynamic Configuration**: Use flexible YAML-based configs for customizable actions.

Beekeeper simplifies managing and testing Bee nodes, whether deployed in Kubernetes or standalone environments.

## Requirements

- Kubernetes cluster (v1.31+)
- [Geth Swap node](https://github.com/ethersphere/helm/tree/master/charts/geth-swap)

## Installation

```bash
make binary
cp dist/beekeeper /usr/local/bin/beekeeper
```

## Run unit tests

Runs available unit tests in pkg folder

```bash
make test
```

## Configuration

Beekeeper is configured with:

- **config file** - sets Beekeeper's internals, and
- **config directory** - sets Bee cluster configuration, checks and simulations

## Config file

Config file is used to set Beekeeper internals:

- config directory location
- Kubernetes client
- Swap client

Default location for config file is: **$HOME/.beekeeper.yaml**

Location can also be set with **--config** flag.

example:

```yaml
config-dir: <user home dir>/.beekeeper/
config-git-repo: ""
config-git-branch: main
config-git-username: <GitHub username>
config-git-password: <GitHub Personal Access Token>
enable-k8s: true
in-cluster: false
kubeconfig: "~/.kube/config"
geth-url: http://geth-swap.geth-swap.dai.internal
bzz-token-address: 0x6aab14fe9cccd64a502d23842d916eb5321c26e7 
eth-account: 0x62cab2b3b55f341f10348720ca18063cdb779ad5
log-verbosity: "info"
loki-endpoint: http://loki.testnet.internal/loki/api/v1/push
```

Beekeeper reads *config-dir* from a local machine by default, but it also supports reading *config-dir* from a Git repo. If field *config-git-repo* is set, it will override *config-dir* and configuration will be read from a Git repo.

If *config-dir* is kept in a Git repo, field *config-git-repo* should point to it, along with *config-git-branch* specifying proper branch. Fields *config-git-username* and *config-git-password* can be set when repo is private.

Official GitHub repository with Beekeeper's configuration is **<https://github.com/ethersphere/beekeeper-config>**

General Notes:

- command flags can be also set through the config file
- k8s client can be disabled with *enable-k8s* flag (default is true)

## Config directory

Config directory is used to group configuration (.yaml) files describing:

- Bee cluster setup in Kubernetes
- checks (integration tests), and
- simulations

Default location for config dir is: **$HOME/.beekeeper/**

Location can also be set with **--config-dir** flag.

Examples of .yaml files can be found in the [Beekeeper repo](https://github.com/ethersphere/beekeeper/tree/master/config).

Config dir's .yaml files have several main blocks:

- **clusters** - defines clusters Beekeeper works with
- **node-groups** - defines Bee node groups that are part of the cluster. Node group is a collection of Bee nodes sharing same configuration parameters.
- **bee-configs** - defines Bee configuration that can be assigned to node-groups
- **checks** - defines checks Beekeeper can execute against the cluster
- **simulations** - defines simulations Beekeeper can execute against the cluster
- **stages** - defines stages for dynamic execution of checks and simulations

### Inheritance

Inheritance can be set through the field *_inherit*.

Clusters, node-groups and bee-configs blocks support inheritance.

example:

```yaml
bee-configs:
  light-node:
    _inherit: default
    full-node: false
```

This setting means that *light-node* bee-config will inherit all parameters from the *default* bee-config, overriding only *full-node* parameter.

### Action types

Action types can be set in every check or simulation definition.

Action types allow defining same check or simulation with different parameters.

example:

```yaml
checks:
  pushsync-chunks:
    options:
      chunks-per-node: 1
      metrics-enabled:
      mode: chunks
      postage-amount: 1000
      postage-depth: 16
      postage-wait: 5s
      retries: 5
      retry-delay: 1s
      upload-node-count: 1
    timeout: 5m
    type: pushsync
  pushsync-light-chunks:
    options:
      chunks-per-node: 1
      metrics-enabled:
      mode: light-chunks
      postage-amount: 1000
      postage-depth: 16
      postage-wait: 5s
      retries: 5
      retry-delay: 1s
      upload-node-count: 1
    timeout: 5m
    type: pushsync
```

This setting means that pushsync check can be executed choosing *pushsync-chunks* or *pushsync-light-chunks* variation.

## Usage

**beekeeper** has following commands:

|command|description|
|-------|-----------|
| check | runs integration tests on a Bee cluster |
| create | creates Bee infrastructure |
| delete | Delete Bee infrastructure |
| fund | Fund Ethereum addresses |
| help | Help about any command |
| print | Print information about a Bee cluster |
| simulate | Run simulations on a Bee cluster |
| version | Print version number |
| node-funder | Fund (top up) Bee nodes |
| node-operator | Auto-funds (top up) Bee nodes on deployment. |
| restart | Restart Bee nodes in Kubernetes |
| stamper | Manage postage batches for nodes |

### check

Command **check** runs ingegration tests on a Bee cluster.

It has following flags:

```console
--checks strings                  list of checks to execute (default [pingpong])
--cluster-name string             cluster name (default "default")
--create-cluster                  creates cluster before executing checks
--help                            help for check
--metrics-enabled                 enable metrics
--metrics-pusher-address string   prometheus metrics pusher address (default "pushgateway.staging.internal")
--seed int                        seed, -1 for random (default -1)
--timeout duration                timeout (default 30m0s)
```

example:

```bash
beekeeper check --checks=pingpong,pushsync
```

### create

Command **create** creates Bee infrastructure. It has two subcommands:

- bee-cluster - creates Bee cluster

It has following flags:

```console
--cluster-name string   cluster name (default "default")
--help                  help for bee-cluster
--timeout duration      timeout (default 30m0s)
--wallet-key string     Hex-encoded private key for the Bee node wallet. Required.
```

It is required to specify *geth-url* and *wallet-key* flags for funding Bee nodes with usage of flags or config file.

example:

```bash
beekeeper create bee-cluster --cluster-name=default
```

- k8s-namespace - creates Kubernetes namespace

example:

```bash
beekeeper create k8s-namespace beekeeper
```

### delete

Command **delete** deletes Bee infrastructure. It has two subcommands:

- bee-cluster - deletes Bee cluster

It has following flags:

```console
--cluster-name string   cluster name (default "default")
--help                  help for bee-cluster
--timeout duration      timeout (default 15m0s)
--with-storage          delete storage
```

example:

```bash
beekeeper delete bee-cluster default
```

- k8s-namespace - deletes Kubernetes namespace

example:

```bash
beekeeper delete k8s-namespace beekeeper
```

### fund

Command **fund** makes BZZ tokens and ETH deposits to given Ethereum addresses.

It has the following flags:

```console
--addresses strings          Bee node Ethereum addresses (must start with 0x)
--address-create             if enabled, creates Ethereum address(es)
--address-count              number of Ethereum addresses to create
--bzz-deposit float          BZZ tokens amount to deposit
--bzz-token-address string   BZZ token address (default "0x6aab14fe9cccd64a502d23842d916eb5321c26e7")
--eth-account string         ETH account address (default "0x62cab2b3b55f341f10348720ca18063cdb779ad5")
--eth-deposit float          ETH amount to deposit
--help                       help for fund
--password                   password for generating Ethereum addresses (default "beekeeper")
--timeout duration           timeout (default 5m0s)
```

examples:

```bash
beekeeper fund --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0 --bzz-deposit 100 --eth-deposit 0.01

beekeeper fund --address-create --address-count 2 --bzz-deposit 100 --eth-deposit 0.01
```

### print

Command **print** prints information about a Bee cluster.

It has following flags:

```console
--cluster-name string   cluster name (default "default")
--help                  help for print
--timeout duration      timeout (default 15m0s)
```

example:

```bash
beekeeper print overlays
```

### simulate

Command **simulate** runs simulations on a Bee cluster.

It has following flags:

```console
--cluster-name string             cluster name (default "default")
--create-cluster                  creates cluster before executing simulations
--help                            help for check
--metrics-enabled                 enable metrics
--metrics-pusher-address string   prometheus metrics pusher address (default "pushgateway.staging.internal")
--seed int                        seed, -1 for random (default -1)
--simulations strings             list of simulations to execute (default [upload])
--timeout duration                timeout (default 30m0s)
```

example:

```bash
beekeeper simulate --simulations=upload
```

### version

Command **version** prints version number.

example:

```bash
beekeeper version
```

### node-funder

Command **node-funder** uses <https://github.com/ethersphere/node-funder> tool to fund (top up) bee nodes up to the specified amount. It can fund all nodes in k8s namespace or it can fund only specified addresses.

It has following flags:

```console
--addresses strings         Comma-separated list of Bee node addresses (must start with 0x). Overrides namespace and cluster name.
--cluster-name string       Name of the Beekeeper cluster to target. Ignored if a namespace is specified.
--help                      help for node-funder
--label-selector string     Kubernetes label selector for filtering resources within the specified namespace. Use an empty string to select all resources. (default "beekeeper.ethswarm.org/node-funder=true")
--min-native float          Minimum amount of chain native coins (xDAI) nodes should have.
--min-swarm float           Minimum amount of swarm tokens (xBZZ) nodes should have.
--namespace string          Kubernetes namespace. Overrides cluster name if set.
--periodic-check duration   Periodic execution check interval.
--timeout duration          Timeout. (default 5m0s)
--wallet-key string         Hex-encoded private key for the Bee node wallet. Required.
```

#### Fund specific addresses

```bash
beekeeper node-funder --geth-url="http://geth-swap.default.testnet.internal" --wallet-key="4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d" --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0 --min-swarm=180 --min-native=2.2
```

#### Fund K8S namespace (use label selector to filter nodes)

```bash
beekeeper node-funder --geth-url="http://geth-swap.default.testnet.internal" --wallet-key="4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d" --namespace=default --min-swarm=180 --min-native=2.2 --label-selector="app=bee"
```

#### Fund all nodes in the cluster (beekeeper configuration)

```bash
beekeeper node-funder --geth-url="http://geth-swap.default.testnet.internal" --wallet-key="4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d" --cluster-name=default --min-swarm=180 --min-native=2.2
```

### node-operator

Command **node-operator** uses <https://github.com/ethersphere/node-funder> tool to fund (top up) bee nodes up to the specified amount. It is running in the Kubernetes namespace and it is watching for Bee node deployments. When new deployment is created, it will fund it with the specified amount. It uses filter "app.kubernetes.io/name=bee" on label to determine which deployments to watch.

It has following flags:

```console
--help                    help for node-operator
--min-native float        Minimum amount of chain native coins (xDAI) nodes should have.
--min-swarm float         Minimum amount of swarm tokens (xBZZ) nodes should have.
--namespace string        Kubernetes namespace to scan for scheduled pods.
--label-selector string   Kubernetes label selector for filtering resources within the specified namespace. An empty string disables filtering, allowing all resources to be selected.
--timeout duration        Operation timeout (e.g., 5s, 10m, 1.5h). Default is 0, which means no timeout.
--wallet-key string       Hex-encoded private key for the Bee node wallet. Required.
```

example:

```bash
beekeeper node-operator --geth-url="http://geth-swap.default.testnet.internal" --wallet-key="4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d" --namespace=default --min-swarm=180 --min-native=2.2 --log-verbosity=3
```

### restart

Command **restart** restarts bee node in Kubernetes , with optional targeting by namespace, label selectors, and node groups.

It has following flags:

```console
--cluster-name string     Kubernetes cluster to operate on (overrides namespace and label selector).
--namespace string        Namespace to delete pods from (only used if cluster name is not set).
--label-selector string   Label selector for resources in the namespace (only used with namespace). An empty string disables filtering, allowing all resources to be selected.
--image string            Container image to use when restarting pods (defaults to current image if not set).
--node-groups strings     Comma-separated list of node groups to target for restarts (applies to all groups if not set).
--timeout duration        Operation timeout (e.g., 5s, 10m, 1.5h). (default 5m0s)
```

example:

```bash
beekeeper restart --cluster-name=default --image="bee:latest" --node-groups="group1,group2" --timeout=10m
```

or

```bash
beekeeper restart -namespace=default --label-selector="app=bee" --timeout=10m
```

### stamper

Command **stamper** manage postage batches for nodes.

General Notes:

- `namespace` or `cluster-name` must be specified to locate the bee nodes.
- If both are provided, `namespace` takes precedence.
- When `namespace` is set, you can use a `label-selector` to filter specific nodes.
- Use `batch-ids` to target specific postage batches, but this is applied after finding/filtering nodes. If `batch-ids` is not provided, all batches in the filtered nodes are targeted.
- If `timeout` is set to 0 and `periodic-check` is bigger than 0, the operation will run indefinitely with periodic checks.

It has following subcommands:

- **create** - creates a postage batch for selected nodes

  It has following flags:

  ```console
  --amount uint             Amount of BZZ in PLURS added that the postage batch will have. (default 100000000)
  --cluster-name string     Target Beekeeper cluster name.
  --depth uint16            Batch depth which specifies how many chunks can be signed with the batch. It is a logarithm. Must be higher than default bucket depth (16)
  --help                    help for create
  --label-selector string   Kubernetes label selector for filtering resources (use empty string for all). (default "beekeeper.ethswarm.org/node-funder=true")
  --namespace string        Kubernetes namespace (overrides cluster name).
  --timeout duration        Operation timeout (e.g., 5s, 10m, 1.5h). (default 5m0s)
  ```

  example:
  
  ```bash
  beekeeper stamper create --cluster-name=default --amount=1000 --depth=16 --timeout=5m
  ```

  or

  ```bash
  beekeeper stamper create --namespace=default --label-selector="app=bee" --amount=1000 --depth=16 --timeout=5m
  ```

- **topup** - tops up postage batch for selected nodes

  It has following flags:

  ```console
  --batch-ids strings         Comma separated list of postage batch IDs to top up. If not provided, all batches are topped up.
  --cluster-name string       Target Beekeeper cluster name.
  --help                      help for topup
  --label-selector string     Kubernetes label selector for filtering resources (use empty string for all). (default "beekeeper.ethswarm.org/node-funder=true")
  --namespace string          Kubernetes namespace (overrides cluster name).
  --periodic-check duration   Periodic check interval. Default is 0, which means no periodic check.
  --timeout duration          Operation timeout (e.g., 5s, 10m, 1.5h). (default 5m0s)
  --topup-to duration         Duration to top up the TTL of a stamp to. (default 720h0m0s)
  --ttl-threshold duration    Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value. (default 120h0m0s)
  ```

  example:
  
  ```bash
  beekeeper stamper topup --cluster-name=default --topup-to=720h --ttl-threshold=120h --periodic-check=1h --timeout=24h
  ```

  or

  ```bash
  beekeeper stamper topup --namespace=default --label-selector="app=bee" --topup-to=720h --ttl-threshold=120h --periodic-check=1h --timeout=24h
  ```

- **dilute** - dilutes postage batch for selected nodes

  It has following flags:

  ```console
  --batch-ids strings         Comma separated list of postage batch IDs to dilute. If not provided, all batches are diluted.
  --cluster-name string       Target Beekeeper cluster name.
  --dilution-depth uint8      Number of levels by which to increase the depth of a stamp during dilution. (default 1)
  --help                      help for dilute
  --label-selector string     Kubernetes label selector for filtering resources (use empty string for all). (default "beekeeper.ethswarm.org/node-funder=true")
  --namespace string          Kubernetes namespace (overrides cluster name).
  --periodic-check duration   Periodic check interval. Default is 0, which means no periodic check.
  --timeout duration          Operation timeout (e.g., 5s, 10m, 1.5h). (default 5m0s)
  --usage-threshold float     Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value. (default 90)
  ```

  example:
  
  ```bash
  beekeeper stamper dilute --cluster-name=default --dilution-depth=1 --usage-threshold=90 --periodic-check=1h --timeout=24h
  ```

  or

  ```bash
  beekeeper stamper dilute --namespace=default --label-selector="app=bee" --dilution-depth=1 --usage-threshold=90 --periodic-check=1h --timeout=24h
  ```

- **set** - sets postage batch for selected nodes

  It has following flags:

  ```console
  --batch-ids strings         Comma separated list of postage batch IDs to set. If not provided, all batches are set.
  --cluster-name string       Target Beekeeper cluster name.
  --dilution-depth uint16     Number of levels by which to increase the depth of a stamp during dilution. (default 1)
  --help                      help for set
  --label-selector string     Kubernetes label selector for filtering resources (use empty string for all). (default "beekeeper.ethswarm.org/node-funder=true")
  --namespace string          Kubernetes namespace (overrides cluster name).
  --periodic-check duration   Periodic check interval. Default is 0, which means no periodic check.
  --timeout duration          Operation timeout (e.g., 5s, 10m, 1.5h). (default 5m0s)
  --topup-to duration         Duration to top up the TTL of a stamp to. (default 720h0m0s)
  --ttl-threshold duration    Threshold for the remaining TTL of a stamp. Actions are triggered when TTL drops below this value. (default 120h0m0s)
  --usage-threshold float     Percentage threshold for stamp utilization. Triggers dilution when usage exceeds this value. (default 90)
  ```

  example:
  
  ```bash
  beekeeper stamper set --cluster-name=default --dilution-depth=1 --usage-threshold=90 --ttl-threshold=120h --topup-to=720h --periodic-check=1h --timeout=24h
  ```

  or

  ```bash
  beekeeper stamper set --namespace=default --label-selector="app=bee" --dilution-depth=1 --usage-threshold=90 --ttl-threshold=120h --topup-to=720h --periodic-check=1h --timeout=24h
  ```

## Global flags

Global flags can be used with any command.

example:

```console
--config string                 Path to the configuration file (default is $HOME/.beekeeper.yaml)
--config-dir string             Directory for configuration files (default "C:\\Users\\ljubi\\.beekeeper")
--config-git-branch string      Git branch to use for configuration files (default "main")
--config-git-dir string         Directory within the Git repository containing configuration files. Defaults to the root directory (default ".")
--config-git-password string    Git password or personal access token for authentication (required for private repositories)
--config-git-repo string        URL of the Git repository containing configuration files (uses the config-dir if not specified)
--config-git-username string    Git username for authentication (required for private repositories)
--enable-k8s                    Enable Kubernetes client functionality (default true)
--geth-url string               URL of the RPC blockchain endpoint
--in-cluster                    Use the in-cluster Kubernetes client
--kubeconfig string             Path to the kubeconfig file (default "~/.kube/config")
--log-verbosity string          Log verbosity level (0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace) (default "info")
--loki-endpoint string          HTTP endpoint for sending logs to Loki (e.g., http://loki.testnet.internal/loki/api/v1/push)
--tracing-enable                Enable tracing for performance monitoring and debugging
--tracing-endpoint string       Endpoint for sending tracing data, specified as host:port (default "127.0.0.1:6831")
--tracing-host string           Host address for sending tracing data
--tracing-port string           Port for sending tracing data
--tracing-service-name string   Service name identifier used in tracing data (default "beekeeper")
```

## Public Testnet Checks

### One by one

```shell
./dist/beekeeper check --cluster-name=bee-testnet --checks=pingpong
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-retrieval
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-settlements
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-manifest
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-pss
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-soc
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-pushsync-chunks
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-postage
./dist/beekeeper check --cluster-name=bee-testnet --checks=pt-gsoc
```

### All at once, sequentially

```shell
./dist/beekeeper check --cluster-name=bee-testnet --timeout=2h --checks=pingpong,pt-retrieval,pt-settlements,pt-manifest,pt-pss,pt-soc,pt-pushsync-chunks,pt-postage,pt-gsoc
```
