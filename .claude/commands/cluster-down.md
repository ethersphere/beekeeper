---
description: Tear down the beekeeper Bee cluster, optionally destroying the whole beelocal substrate
argument-hint: [cluster-name] [--destroy] [--bee-repo <path>]
---

Tear down the local Bee test cluster. Cluster name: "$1" if given, otherwise
`local-dns`. This is destructive and local-only.

Before removing anything, report what is actually running (`k3d cluster list`,
`kubectl get pods -n local`) and say what you are about to remove. Decide the scope
from "$ARGUMENTS":

1. **Delete the Bee cluster** (default) — from this repo:

   ```
   ./dist/beekeeper delete bee-cluster --cluster-name=<name>
   ```

   This removes the Bee nodes but leaves the k3d cluster, registry and geth-swap
   running, so `/cluster-up` can redeploy quickly.

2. **Destroy the substrate** — only when `--destroy` was passed. After the delete
   above (or if the cluster is already gone), from the bee repo — resolved as
   `--bee-repo <path>`, else `$BEE_REPO`, else
   `$(git rev-parse --show-toplevel)/../bee`:

   ```
   make beelocal OPTS='skip-vet' ACTION=destroy
   ```

   This deletes the k3d cluster and the local registry. The next `/beelocal-up` will
   have to prepare everything from scratch.

Echo each command before running it. Do not touch git.
