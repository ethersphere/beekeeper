# Ethereum Swarm Beekeeper

[![Go](https://github.com/ethersphere/beekeeper/workflows/Go/badge.svg)](https://github.com/ethersphere/beekeeper/actions)

**Ethereum Swarm Beekeeper** is tool used for orchestrating cluster of [Ethereum Swarm Bee](https://github.com/ethersphere/bee) and running integration tests and simulations against it in the Kubernetes.

# Requirements

* Kubernetes cluster (v1.19+)
* [Geth Swap node](https://github.com/ethersphere/helm/tree/master/charts/geth-swap)

# Installation

```bash
make binary
cp dist/beekeeper /usr/local/bin/beekeeper
```

# Configuration

Beekeeper is configured with:
* **config file** - sets Beekeeper's internals, and
* **config directory** - sets Bee cluster configuration, checks and simulations

## Config file

Config file is used to set Beekeeper internals:
* config directory location
* Kubernetes client
* Swap client

Default location for config file is: **$HOME/.beekeeper.yaml**

Location can also be set with **--config** flag.

example:
```
config-dir: <user home dir>/.beekeeper/
enable-k8s: true
in-cluster: false
kubeconfig: "~/.kube/config"
geth-url: http://geth-swap.geth-swap.dai.internal
bzz-token-address: 0x6aab14fe9cccd64a502d23842d916eb5321c26e7 
eth-account: 0x62cab2b3b55f341f10348720ca18063cdb779ad5
```

NOTE: command flags can be also set through the config file

## Config directory

Config directory is used to group configuration (.yaml) files describing:
* Bee cluster setup in Kubernetes
* checks (integration tests), and
* simulations

Default location for config dir is: **$HOME/.beekeeper/**

Location can also be set with **--config-dir** flag.

Examples of .yaml files can be found in the [Beekeeper repo](https://github.com/ethersphere/beekeeper/tree/master/config).

Config dir's .yaml files have several main blocks:
* **clusters** - defines clusters Beekeeper works with
* **node-groups** - defines Bee node groups that are part of the cluster. Node group is a collection of Bee nodes sharing same configuration parameters.
* **bee-configs** - defines Bee configuration that can be assigned to node-groups
* **checks** - defines checks Beekeeper can execute against the cluster
* **simulations** - defines simulations Beekeeper can execute against the cluster
* **stages** - defines stages for dynamic execution of checks and simulations

### Inheritance

Inheritance can be set through the field *_inherit*.

Clusters, node-groups and bee-configs blocks support inheritance. 

example:
```
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
```
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

# Usage

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

## check

Command **check** runs ingegration tests on a Bee cluster.

It has following flags:

```
--checks strings        list of checks to execute (default [pingpong])
--cluster-name string   cluster name (default "default")
--create-cluster        creates cluster before executing checks
--help                  help for check
--metrics-enabled       enable metrics
--seed int              seed, -1 for random (default -1)
--timeout duration      timeout (default 30m0s)
--with-funding          fund nodes (default false)
```

example:
```
beekeeper check --checks=pingpong,pushsync
```

## create

Command **create** creates Bee infrastructure. It has two subcommands:
* bee-cluster - creates Bee cluster

    It has following flags:

    ```
    --cluster-name string   cluster name (default "default")
    --help                  help for bee-cluster
    --timeout duration      timeout (default 30m0s)
    --with-funding          fund nodes (default true)
    ```

    example:
    ```
    beekeeper create bee-cluster default
    ```

* k8s-namespace - creates Kubernetes namespace

    example:
    ```
    beekeeper create k8s-namespace beekeeper
    ```

## delete

Command **delete** deletes Bee infrastructure. It has two subcommands:
* bee-cluster - deletes Bee cluster

    It has following flags:

    ```
    --cluster-name string   cluster name (default "default")
    --help                  help for bee-cluster
    --timeout duration      timeout (default 15m0s)
    --with-storage          delete storage
    ```

    example:
    ```
    beekeeper delete bee-cluster default
    ```

* k8s-namespace - deletes Kubernetes namespace

    example:
    ```
    beekeeper delete k8s-namespace beekeeper
    ```

## fund

Command **fund** makes BZZ tokens and ETH deposits to given Ethereum addresses.

It has the following flags:

```
--addresses strings          Bee node Ethereum addresses (must start with 0x)
--bzz-deposit float          BZZ tokens amount to deposit
--bzz-token-address string   BZZ token address (default "0x6aab14fe9cccd64a502d23842d916eb5321c26e7")
--eth-account string         ETH account address (default "0x62cab2b3b55f341f10348720ca18063cdb779ad5")
--eth-deposit float          ETH amount to deposit
--geth-url string            Geth node URL (default "http://geth-swap.geth-swap.dai.internal")
--help                       help for fund
--timeout duration           timeout (default 5m0s)
```

example:
```
beekeeper fund --addresses=0xf176839c150e52fe30e5c2b5c648465c6fdfa532,0xebe269e07161c68a942a3a7fce6b4ed66867d6f0 --eth-deposit 0.01 --bzz-deposit 100
```

## print

Command **print** prints information about a Bee cluster.

It has following flags:

```
--cluster-name string   cluster name (default "default")
--help                  help for print
--timeout duration      timeout (default 15m0s)
```

example:
```
beekeeper print overlays
```

## simulate

Command **simulate** runs simulations on a Bee cluster.

It has following flags:

```
--cluster-name string   cluster name (default "default")
--create-cluster        creates cluster before executing simulations
--help                  help for check
--metrics-enabled       enable metrics
--seed int              seed, -1 for random (default -1)
--simulations strings   list of simulations to execute (default [upload])
--timeout duration      timeout (default 30m0s)
--with-funding          fund nodes (default false)
```

example:
```
beekeeper simulate --simulations=upload
```

## version

Command **version** prints version number.

example:
```
beekeeper version
```
