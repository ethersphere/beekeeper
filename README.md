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
| peercount | Check node's peer count for all nodes in the cluster |
| pingpong | Checks pingpong |
| pushsync | Checks push sync |

### fullconnectivity

**fullconnectivity** checks full connectivity in the cluster.

Example:
```bash
beekeeper check fullconnectivity --node-count 3 --namespace bee
```
 or, shorthand:
 ```bash
beekeeper check fullconnectivity -c 3 -n bee
```

### peercount

**peercount** checks node's peer count for all nodes in the cluster.
It retrieves list of peers from node's Debug API (/peers endpoint).

Example:
```bash
beekeeper check peercount --node-count 3 --namespace bee
```
 or, shorthand:
 ```bash
beekeeper check peercount -c 3 -n bee
```

### pingpong

**pingpong** checks pingpong.

Example:
```bash
beekeeper check pingpong --node-count 3 --namespace bee
```
 or, shorthand:
 ```bash
beekeeper check pingpong -c 3 -n bee
```

### pushsync

**pushsync** checks push-sync.

Example:
```bash
beekeeper check pushsync --node-count 3 --namespace bee
```
 or, shorthand:
 ```bash
beekeeper check pushsync -c 3 -n bee
```
