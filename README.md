# Ethereum Swarm Beekeeper

[![Go](https://github.com/ethersphere/beekeeper/workflows/Go/badge.svg)](https://github.com/ethersphere/beekeeper/actions)

**Ethereum Swarm Beekeeper** is tool used for testing of [Ethereum Swarm Bee](https://github.com/ethersphere/bee).

# Installing

```bash
make binary
cp dist/beekeeper /usr/local/bin/beekeeper
```

# Usage

**beekeeper** has following commands:

|command|description|
|-------|-----------|
| check | Run tests on Bee node(s) |
| help | Help about any command |
| print | Print Bee cluster info |
| version | Print version number |

## check

Command **check** runs test(s) on Bee node(s).
 Each test is implemented as a subcommand.

Available subcommands:

|subcommand|description|
|----------|-----------|
| fileretrieval | Checks file retrieval ability of the cluster |
| fileretrievaldynamic | Checks file retrieval ability of the dynamic cluster |
| fullconnectivity | Checks full connectivity in the cluster |
| kademlia | Checks Kademlia topology in the cluster |
| localpinning | Checks local pinning ability of the cluster |
| peercount | Count peers for all nodes in the cluster |
| pingpong | Executes ping from all nodes to all other nodes in the cluster |
| pullsync | Checks pullsync ability of the cluster |
| pushsync | Checks pushsync ability of the cluster |
| retrieval | Checks retrieval ability of the cluster |

### fileretrieval

**fileretrieval** checks file retrieval ability of the cluster.
It uploads given number of files to given number of nodes, 
and attempts retrieval of those files from the last node in the cluster.

Example:
```bash
beekeeper check fileretrieval --namespace bee --node-count 3 --upload-node-count 2 --files-per-node 4 --file-size 1048576
```

### fileretrievaldynamic

**fileretrievaldynamic** checks file retrieval ability of the dynamic cluster. It uploads file to a random node, than it downloads that file from given number of random nodes. Than it stops given number of other random nodes, and tries to download file again from other random nodes. It starts stopped nodes and downloads file from them.
It has an option to add new nodes to the cluster and repeat previous steps with the updated cluster.

Example:
```bash
beekeeper check fileretrievaldynamic --namespace bee --node-count 5 --new-node-count 3 --kubeconfig /Users/<user>/.kube/config 
```

### fullconnectivity

**fullconnectivity** checks if every node has connectivity to all other nodes in the cluster.

Example:
```bash
beekeeper check fullconnectivity --namespace bee --node-count 3
```

### kademlia

**kademlia** checks Kademlia topology in the cluster

Example:
```bash
beekeeper check kademlia --namespace bee --node-count 3
```

### localpinning

**localpinning** checks local pinning ability of the cluster.

Example:
```bash
beekeeper check localpinning --namespace bee --node-count 3
```

### peercount

**peercount** counts peers for all nodes in the cluster.

Example:
```bash
beekeeper check peercount --namespace bee --node-count 3
```

### pingpong

**pingpong** executes ping from all nodes to all other nodes in the cluster,
and prints round-trip time (RTT) of each ping.

Example:
```bash
beekeeper check pingpong --namespace bee --node-count 3
```

### pullsync

**pullsync** checks pullsync ability of the cluster.

Example:
```bash
beekeeper check pullsync --namespace bee --node-count 3
```

### pushsync

**pushsync** checks pushsync ability of the cluster.
It uploads given number of chunks to given number of nodes, and checks if chunks are synced to their closest nodes.

Example:
```bash
beekeeper check pushsync --namespace bee --node-count 3 --upload-node-count 2 --chunks-per-node 4
```

### retrieval

**retrieval** checks retrieval ability of the cluster.
It uploads given number of chunks to given number of nodes, 
and attempts retrieval of those chunks from the last node in the cluster.

Example:
```bash
beekeeper check retrieval --namespace bee --node-count 3 --upload-node-count 2 --chunks-per-node 4
```

## print

Command **print** prints info about Bee cluster.
 Each type of information is implemented as a subcommand.

Available subcommands:

|subcommand|description|
|----------|-----------|
| addresses | Print addresses for every node in a cluster |
| overlays | Print overlay address for every node in a cluster |
| peers | Print list of peers for every node in a cluster |
| topologies | Print list of Kademlia topology for every node in a cluster |
