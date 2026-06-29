# Bee pull-sync halt — reproduction & findings

**The halt:** after a populated neighbourhood is disrupted, stock-master pull-sync cannot re-converge —
affected nodes stay permanently un-synced and, being staked, can no longer win redistribution rounds.
Reproduced locally on the `local-dns` cluster against **A = stock master + radius patches**.

**Status:** mechanism + pull-sync non-convergence reproduced and characterized across 4 runs, incl. a
fully-staked run (2026-06-26) that pins down the **staked round-loss**: every survivor hits
`is_playing_errors`, and a node that cannot finish the re-sync cannot produce a `ReserveSample`, so it
loses every round. Remaining: the **A/B** confirmation against **B = `pullsync-optimal-design`**.

**Automation:** this manual recipe is now folded into the `reserveradius` check's `mode: halt` (and the
`observe`+disrupt shape) — stake → drive radius → disrupt → observe/classify/verdict, unattended. See
the check plan [`docs/radius-halt-check-plan.md`](radius-halt-check-plan.md) and the check
[`pkg/check/reserveradius/README.md`](../pkg/check/reserveradius/README.md); the `ci-radius-halt` and
`ci-reserve-radius-observe-disrupt` entries live in `config/local.yaml`.

## Mechanism

The storage radius must be **≥3** to reproduce the halt. At radius 1–2 on a small cluster the reserve is
capacity-driven and `reserveSizeWithinRadius` is degenerate (**0**) — there is no real neighbourhood
data, so nothing can stall. At radius 3 each node holds real in-radius data (~2200 chunks). Removing
nodes from that neighbourhood drops redundancy below what the survivors can re-replicate; the stock
puller **stops delivering before convergence**, so the nodes never re-sync. It is not a crash or a hard
freeze — it is pull-sync silently failing to converge.

Two ways to disrupt the neighbourhood:

- **Node removal (reliable).** Remove nodes from a populated radius-3 neighbourhood. The halt does
  **not** require a radius decrease — disruption at a populated radius is enough, and this is the
  deterministic lever used below.
- **Radius decrease (the faithful October trigger, intermittent).** A decrease *expands* the
  neighbourhood (radius N covers half the keyspace of N+1), so a node becomes responsible for far more
  chunks and must pull-sync them at once. But the decrease is **gated** (`pkg/storer/reserve.go`): it
  fires only when `countWithinRadius < threshold(capacity)` **AND** `syncer.SyncRate()==0` **AND**
  `radius > minimumRadius`. Stock `threshold = capacity*50%`; `radius_threshold.patch` raises it to
  `capacity`. `SyncRate()==0` rarely holds on a busy cluster, so decreases are slow/intermittent —
  node scale-down triggered one only 1 of 4 attempts; stop-load and `dilute` never did (`dilute` is the
  *wrong* lever — it raises committed capacity). The faithful decrease lever is a commitment drop
  (batch expiry), which is slow/hard to control locally.

## Prerequisites

- beelocal/k3d substrate + `geth-swap` + metrics stack (pushgateway 9091 / prometheus 9090 / grafana 3000).
- Patched image `k3d-registry.localhost:5000/ethersphere/bee:latest` = `radius_reserve.patch`
  (DefaultReserveCapacity 4000, ReserveWakeUpDuration 10s) + `radius_threshold.patch` (threshold = capacity).
  Build it (apply both patches to the bee source, build, push, revert source):

  ```
  patch pkg/storer/storer.go  .github/patches/radius_reserve.patch
  patch pkg/storer/reserve.go .github/patches/radius_threshold.patch
  make docker-build PLATFORM=linux/arm64 BEE_IMAGE=k3d-registry.localhost:5000/ethersphere/bee:latest \
    REACHABILITY_OVERRIDE_PUBLIC=true BATCHFACTOR_OVERRIDE_PUBLIC=2
  docker push k3d-registry.localhost:5000/ethersphere/bee:latest
  git checkout pkg/storer/storer.go pkg/storer/reserve.go
  ```

- `config/local.yaml`: `local-dns` bee `count: 6`, light `count: 0`; `ci-load-soak` `max-committed-depth: 3`.
- Geth **chain price must be non-zero** (e.g. 24000); price 0 logs `invalid chain price` and breaks postage.
- Env prefix for every beekeeper command (local chain; `~/.beekeeper.yaml` is Sepolia):
  `BEEKEEPER_GETH_URL=http://geth-swap.localhost BEEKEEPER_BZZ_TOKEN_ADDRESS=0x6aab14fe9cccd64a502d23842d916eb5321c26e7 BEEKEEPER_ETH_ACCOUNT=0x62cab2b3b55f341f10348720ca18063cdb779ad5 BEEKEEPER_WALLET_KEY=4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d`

## Reproducible steps

1. **Recreate a clean cluster** (NEVER `kubectl rollout restart` — storage is emptyDir; a restart wipes
   reserve+statestore+chequebook and needs re-funding). Geth survives (deployed separately):

   ```
   beekeeper delete bee-cluster --cluster-name=local-dns --geth-url http://geth-swap.localhost
   beekeeper create bee-cluster --cluster-name=local-dns --geth-url http://geth-swap.localhost --wallet-key <WALLET_KEY>
   ```

2. **Stake every node** via the API (minimum `1e17`; height 0 because `reserve-capacity-doubling=0`).
   Staking is not needed for the *sync* signals, but it is needed to measure the redistribution
   round-loss. NOTE: `ci-stake` does NOT stake the cluster — it is a single-node contract test that
   deposits then withdraws to 0.

   ```
   for n in bee-0 bee-1 bee-2 bee-3 bee-4 bee-5; do
     curl -s -XPOST http://$n.localhost/stake/100000000000000000
   done
   # verify: curl -s http://bee-0.localhost/stake | jq .stakedAmount  → 100000000000000000 on all six
   ```

3. **Drive the radius to 3** (stop the load once every node reports `storageRadius=3`):

   ```
   beekeeper check --cluster-name=local-dns --checks=ci-load-soak --metrics-enabled=true --metrics-pusher-address=localhost:9091
   pkill -f 'beekeeper check'        # reserveSizeWithinRadius is now REAL (~2000+), not 0
   ```

4. **Trigger the halt — remove 2 of the 6 nodes** (disrupt the populated neighbourhood):

   ```
   kubectl delete statefulset bee-4 bee-5 -n local
   kubectl delete pod bee-4-0 bee-5-0 -n local --grace-period=3
   ```

5. **Observe the 4 survivors for ≥25 min** (onset is delayed ~3–9 min — short windows miss it):

   ```
   for n in bee-0 bee-1 bee-2 bee-3; do
     curl -s http://$n.localhost/status            | jq '{r:.storageRadius,w:.reserveSizeWithinRadius,ps:.pullsyncRate}'
     curl -s http://$n.localhost/redistributionstate | jq '{fs:.isFullySynced,fr:.isFrozen,rnd:.round}'
     curl -s http://$n.localhost/metrics | grep '^bee_pullsync_chunks_delivered '
   done
   ```

6. **Cleanup:** `beekeeper delete bee-cluster --cluster-name=local-dns --geth-url http://geth-swap.localhost`.

## What happens (measured; consistent across runs, numbers from the 31-min staked run)

| Phase | t after removal | Observation |
| --- | --- | --- |
| Quiet | 0–~8 min | radius 3, `within_radius=0`, `fullySynced=true`, rounds advance. Delayed, variable onset (~3–9 min). |
| Onset | ~8 min | `within_radius` jumps 0→~2200 on all survivors; some radius 3→2; `pullsyncRate` spikes (0.7–0.9); `fullySynced→false`. |
| Stall | ~9–29 min | all survivors stuck `fullySynced=false`; `pullsyncRate` decays monotonically toward 0; `bee_pullsync_chunks_delivered` **plateaus** after one onset burst while `offered` ran 24k–34k; rounds advance but stuck nodes can't participate. |
| End | ~30 min | **split-brain, never re-converged** — nodes at *different* radii (3/2/2/3) and four `within_radius` plateaus. Some flip *back* to `fullySynced=true` once `pullsyncRate→0`, but `delivered` stayed flat — they "synced" by the **puller giving up** (SyncRate=0) at inconsistent radii, not by completing. No hard `isFrozen`. No recovery in 30 min. |

**Smoking gun:** `bee_pullsync_chunks_delivered` goes flat while `fullySynced=false` — the puller gives
up before completing the re-sync. `offered/delivered` ran ~2.4×–17.7× — SWIP-25's k×-redundancy problem.

## Redistribution round-loss (staked run, 2026-06-26 — the economic impact)

With all six nodes staked, `bee_storageincentives_*` at the stall showed:

- `is_playing_errors = 3` on **all four** survivors — every node errors when it tries to play.
- A node that never finishes the re-sync has `reserve_sample_duration = 0` and `winner = 0` (e.g.
  bee-2) — it **cannot produce a valid `ReserveSample`, so it cannot win any round**.
- Rounds keep advancing (no hard freeze at this scale), so the chain does not pause — staked survivors
  simply **lose every round**. At network scale, with frozen rounds and slashing, this is the October halt.

## Findings

- The radius **increase** is deterministic (committed-depth lever); the **decrease** is not reliably
  stageable on a small cluster (see Mechanism). The halt does not require a decrease.
- On **A** the cluster **never re-converges** after disruption (persistent ≥30 min, no recovery).
- The small-cluster signature is a **soft stall / non-convergence**, not a hard `isFrozen` freeze — but
  staked un-synced survivors still lose every redistribution round, which is the halt at scale.

## Gotchas

- **Radius 3 is required** — radius 1–2 is degenerate (`within_radius=0`, nothing to stall on).
- **Ephemeral storage (emptyDir)** — recreate, don't restart; re-stake after every recreate. If a node
  was restarted and lost its chequebook (`/status` 503, "no chequebook found"), recover with
  `beekeeper node-funder --namespace local --min-native 1 --min-swarm 100`.
- **Staking** — use `POST /stake/<amount>` per node; `ci-stake` does not stake the cluster.
- **Pull-sync must be healthy** — if node logs show `pushsync … context deadline exceeded` cluster-wide,
  chunks won't propagate, the reserve won't fill, and the radius stays 0. Restart the beelocal substrate
  (a cluster recreate alone does not fix it).
- **Chain price must be non-zero** (price 0 breaks postage/uploads).
- **Observe ≥25 min** — the delayed onset means short windows wrongly conclude "no effect".

## A/B proof (next)

Run this identical recipe against **B = `pullsync-optimal-design`** (built with the same `radius_*`
patches) and confirm the survivors re-converge (`fullySynced→true`) where A stays stuck. The staked run
gives the decisive B-side metric: survivors should keep `is_playing_errors`/`winner` healthy because
they finish the re-sync and can still sample. That is the proof the redesign fixes the halt.

Artifacts from the staked run: `radius-drive.csv` (radius 0→3) and `halt-timeline.csv` (survivors, 30 s,
full signal set incl. stake + round + delivered).
