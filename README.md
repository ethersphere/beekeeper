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
- [Global flags](#global-flags)

## Introduction

**Ethereum Swarm Beekeeper** is tool used for orchestrating cluster of [Ethereum Swarm Bee](https://github.com/ethersphere/bee) and running integration tests and simulations against it in the Kubernetes.

## Requirements

- Kubernetes cluster (v1.19+)
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

NOTE: command flags can be also set through the config file

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
--geth-url string       Endpoint to chain node. Required.
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
--geth-url string            Geth node URL (default "http://geth-swap.geth-swap.dai.internal")
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
--addresses strings       Comma-separated list of Bee node addresses (must start with 0x). Overrides namespace and cluster name.
--geth-url string         Endpoint to chain node. Required.
--cluster-name string     Cluster name. Ignored if addresses or namespace are set.
--help                    help for node-funder
--min-native float        Minimum amount of chain native coins (xDAI) nodes should have.
--min-swarm float         Minimum amount of swarm tokens (xBZZ) nodes should have.
--namespace string        Kubernetes namespace. Overrides cluster name if set.
--label-selector string   Kubernetes label selector for filtering resources within the specified namespace. An empty string disables filtering, allowing all resources to be selected.
--timeout duration        Timeout. (default 5m0s)
--wallet-key string       Hex-encoded private key for the Bee node wallet. Required.
```

example:

```bash
beekeeper node-funder --geth-url="http://geth-swap.default.testnet.internal" --wallet-key="4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d" --namespace=default --min-swarm=180 --min-native=2.2 --log-verbosity=3
```

### node-operator

Command **node-operator** uses <https://github.com/ethersphere/node-funder> tool to fund (top up) bee nodes up to the specified amount. It is running in the Kubernetes namespace and it is watching for Bee node deployments. When new deployment is created, it will fund it with the specified amount. It uses filter "app.kubernetes.io/name=bee" on label to determine which deployments to watch.

It has following flags:

```console
--geth-url string         Endpoint to chain node. Required.
--help                    help for node-operator
--min-native float        Minimum amount of chain native coins (xDAI) nodes should have.
--min-swarm float         Minimum amount of swarm tokens (xBZZ) nodes should have.
--namespace string        Kubernetes namespace to scan for scheduled pods.
--label-selector string   Kubernetes label selector for filtering resources within the specified namespace. An empty string disables filtering, allowing all resources to be selected.
--timeout duration        Timeout. Default is infinite.
--wallet-key string       Hex-encoded private key for the Bee node wallet. Required.
```

example:

```bash
beekeeper node-operator --geth-url="http://geth-swap.default.testnet.internal" --wallet-key="4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d" --namespace=default --min-swarm=180 --min-native=2.2 --log-verbosity=3
```

## Global flags

Global flags can be used with any command.

example:

```console
--config string                 config file (default is $HOME/.beekeeper.yaml)
--config-dir string             config directory (default is $HOME/.beekeeper/)
--config-git-branch string      Git branch (default "main")
--config-git-password string    Git password or personal access tokens (needed for private repos)
--config-git-repo string        Git repository with configurations (uses config directory when Git repo is not specified) (default "")
--config-git-username string    Git username (needed for private repos)
--log-verbosity string          log verbosity level 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=trace (default "info")
--loki-endpoint string          loki http endpoint for pushing local logs (use http://loki.testnet.internal/loki/api/v1/push)
--tracing-enable                enable tracing
--tracing-endpoint string       endpoint to send tracing data (default "tempo-tempo-distributed-distributor.observability:6831")
--tracing-host string           host to send tracing data
--tracing-port string           port to send tracing data
--tracing-service-name string   service name identifier for tracing (default "beekeeper")
```
