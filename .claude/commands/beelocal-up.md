---
description: Bring up the beelocal substrate (k3d cluster + local registry + geth-swap) that beekeeper deploys onto
argument-hint: [--with-bee-image] [--bee-repo <path>]
---

Bring up the substrate a local Bee cluster needs: the k3d cluster `bee`, the local
registry, and the `geth-swap` devchain. It stops there — `/cluster-up` deploys the Bee
nodes afterwards.

`make beelocal` lives in the bee repo. Resolve it once, up front — first match wins: a
`--bee-repo <path>` argument, then `$BEE_REPO`, then the sibling checkout the README
quick start produces:

```
BEE_REPO="${BEE_REPO:-$(git rev-parse --show-toplevel)/../bee}"
```

If the result is not a bee checkout, ask for the path instead of guessing.

Everything here is local-only; never run `ACTION=destroy` — that belongs to
`/cluster-down --destroy`.

1. **Check what is already up** and pick the cheapest path:

   ```
   k3d cluster list bee
   curl -s http://k3d-registry.localhost:5000/v2/_catalog
   kubectl get pods -n local -l app.kubernetes.io/name=geth-swap
   ```

   | State | Do |
   |---|---|
   | cluster running, registry + geth up | nothing — report and point at `/cluster-up` |
   | cluster exists, servers stopped | `make beelocal ACTION=start` |
   | no cluster | full prepare (step 3) |

2. **`/etc/hosts`** — beelocal maps `*.localhost` (registry, geth-swap, bee-N,
   bootnode-N, light-N) to `127.0.0.1`. The step is idempotent and already satisfied
   when `grep -q 'swarm bee' /etc/hosts` matches. If it does not match, STOP: it needs
   an interactive sudo. Ask the user to run it themselves from the bee repo (in Claude
   Code, prefix with `!`):

   ```
   make beelocal ACTION=add-hosts
   ```

3. **Prepare** — from `$BEE_REPO`:

   ```
   make beelocal ACTION=prepare SETUP_CONTRACT_IMAGE_TAG=0.9.4 OPTS='skip-local'
   ```

   - `prepare` = check tooling → add-hosts → create k3d cluster + registry → helm
     install `geth-swap` (waits for the `setupcontracts` job to reach `Completed`).
   - `SETUP_CONTRACT_IMAGE_TAG` pins `ethersphere/bee-localchain`; it must match bee
     CI (`.github/workflows/beekeeper.yml`, currently `0.9.4`) or the deployed contract
     addresses differ from what the tooling expects.
   - `OPTS='skip-local'` skips beelocal's own bee build, which is slow and **not
     CI-parity** (no `REACHABILITY_OVERRIDE_PUBLIC`, no `.github/patches`).
     `/cluster-up --rebuild-bee` builds the correct image. Pass `--with-bee-image` in
     "$ARGUMENTS" only if the user explicitly wants beelocal's build — then use
     `OPTS='skip-vet'` and warn that the image is not CI-parity.
   - On an already-running cluster `prepare` falls through to that build, which is why
     step 1 matters.

4. **Verify** — substrate only, there are no Bee nodes yet:
   - `k3d cluster list bee` servers running; `kubectl get nodes` reaches the API.
   - registry catalog answers.
   - `kubectl get pods -n local` → `geth-swap-*` Running, setupcontracts `Completed`.
   - `curl -s -X POST http://geth-swap.localhost -H 'content-type: application/json' --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}'` → `0x3039`.

5. **Report** PASS/FAIL per item and the next command: `/cluster-up local-dns
   --rebuild-bee` on a fresh substrate, or plain `/cluster-up local-dns` if the
   registry already lists `ethersphere/bee`.

Echo each command before running it and surface the tail of any failure instead of
pushing on. Do not commit anything.
