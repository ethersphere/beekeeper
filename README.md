# Ethereum Swarm Beekeeper

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
| version | Print version number |

## check

Command **check** runs test(s) on Bee node(s).
 Each test is implemented as a subcommand.

Available subcommands:

|subcommand|description|
|----------|-----------|
| fullconnectivity | Checks full connectivity in the cluster |
| peercount | Count peers for all nodes in the cluster |
| pingpong | Executes ping from all nodes to all other nodes in the cluster |
| pushsync | Checks pushsync ability of the cluster |

### fullconnectivity

**fullconnectivity** checks if every node has connectivity to all other nodes in the cluster.

Example:
```bash
beekeeper check fullconnectivity --namespace bee --node-count 3
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

### pushsync

**pushsync** checks pushsync ability of the cluster.
It uploads given number of chunks to given number of nodes, and checks if chunks are synced to their closest nodes.

Example:
```bash
beekeeper check pushsync --namespace bee --node-count 3 --upload-node-count 2 --chunks-per-node 4
```
