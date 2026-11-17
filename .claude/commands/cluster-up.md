---
description: Deploy a Bee cluster with beekeeper onto an already-running beelocal substrate
argument-hint: [cluster-name] [--rebuild-bee] [--bee-repo <path>]
---

Deploy a local Bee test cluster with beekeeper. Cluster name: "$1" if given, otherwise
`local-dns` (names are defined in `config/local.yaml`).

Run beekeeper commands from this repo. The bee repo is needed only to build the image.
Resolve it once, up front — first match wins: a `--bee-repo <path>` argument, then
`$BEE_REPO`, then the sibling checkout the README quick start produces:

```
BEE_REPO="${BEE_REPO:-$(git rev-parse --show-toplevel)/../bee}"
```

If the result is not a bee checkout, ask for the path instead of guessing.

The steps are heavy but local-only: check whether each is already satisfied and skip if
so, never tear anything down, and stop and report on failure rather than pushing on.

1. **Substrate precondition (verify only)** — `k3d cluster list` shows a running
   cluster, `curl -s http://k3d-registry.localhost:5000/v2/_catalog` answers, and
   `kubectl get nodes` reaches the API server. If not, STOP and run `/beelocal-up`
   first.

2. **Bee image — must be CI-parity.** Build and push only when `--rebuild-bee` was
   passed in "$ARGUMENTS", the registry catalog has no `ethersphere/bee` yet, or the
   user has bee changes they want tested (`git -C "$BEE_REPO" status --short`).
   Otherwise reuse the registry image. From `$BEE_REPO`:

   ```
   patch pkg/api/postage.go .github/patches/postage_api.patch
   patch pkg/retrieval/retrieval.go .github/patches/retrieval.patch
   make docker-build PLATFORM=linux/$(go env GOARCH) \
     BEE_IMAGE=k3d-registry.localhost:5000/ethersphere/bee:latest \
     REACHABILITY_OVERRIDE_PUBLIC=true BATCHFACTOR_OVERRIDE_PUBLIC=2
   docker push k3d-registry.localhost:5000/ethersphere/bee:latest
   patch -R pkg/api/postage.go .github/patches/postage_api.patch
   patch -R pkg/retrieval/retrieval.go .github/patches/retrieval.patch
   ```

   Always revert both patches and delete any `*.orig`, even if the build fails —
   leaving them applied silently poisons the next bee build.

   All four pieces are what bee CI uses (`.github/workflows/beekeeper.yml`):
   - `REACHABILITY_OVERRIDE_PUBLIC=true` (ldflag, defaults to `false`) — without it
     AutoNAT never resolves inside k3d, `/topology` reports `reachability: Unknown`,
     and the pushsync handler never stores (gate:
     `IsReachable() && Proximity >= storageRadius`). Symptoms: direct uploads take
     ~30s per chunk, "could not push chunk" / "context deadline exceeded" everywhere,
     cross-node downloads fail. Checks look hung rather than failed.
   - `BATCHFACTOR_OVERRIDE_PUBLIC=2` — batch depth factor for a small cluster.
   - `postage_api.patch` — allows batches with depth < 17.
   - `retrieval.patch` — disables multiplexed forwarding, as in CI.

3. **Beekeeper binary** — `make binary` if `./dist/beekeeper` is missing or older than
   recent source changes.

4. **Deploy**:

   ```
   ./dist/beekeeper create bee-cluster --cluster-name=<name> --log-verbosity=debug
   ```

   `create` funds the nodes over the chain configured by `geth-url`,
   `bzz-token-address`, `eth-account` and `wallet-key`. If the active
   `~/.beekeeper.yaml` points at a remote chain (e.g. a testnet), override those four
   for this one command instead of editing the user's global config — Viper reads env
   with prefix `BEEKEEPER_`, `-`→`_`. Do not pass `--config`; the binary's flag parse
   rejects it. The local values are in `config/beekeeper-local.yaml`:

   ```
   BEEKEEPER_GETH_URL=http://geth-swap.localhost \
   BEEKEEPER_BZZ_TOKEN_ADDRESS=0x6aab14fe9cccd64a502d23842d916eb5321c26e7 \
   BEEKEEPER_ETH_ACCOUNT=0x62cab2b3b55f341f10348720ca18063cdb779ad5 \
   BEEKEEPER_WALLET_KEY=4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d \
   ./dist/beekeeper create bee-cluster --cluster-name=<name> --log-verbosity=debug
   ```

   These are the beelocal devchain defaults (public ci-stake key, chainId `0x3039`) —
   local use only. A good run logs `fund options, eth: 0.1, bzz: 100`.

5. **Verify** — run `/cluster-verify <name>`, then report PASS/FAIL and the next
   command, e.g.
   `./dist/beekeeper check --cluster-name=<name> --checks=ci-pingpong --log-verbosity=debug`.
   Long checks need an explicit `--timeout`: it defaults to 30m and bounds the whole
   run, not each check.

Echo each command before running it and surface the tail of any failing output. Do not
commit anything.
