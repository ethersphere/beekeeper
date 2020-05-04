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
| peercount | Check node's peer count for all nodes in the cluster |

### peercount

**peercount** checks node's peer count for all nodes in the cluster.
It retrieves list of peers from node's Debug API (/peers endpoint).

Example:
```bash
beekeeper --node-count 3 --namespace bee
```
 or, shorthand:
 ```bash
beekeeper -c 3 -n bee
```
