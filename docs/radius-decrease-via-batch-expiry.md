# Provoking a radius decrease via postage-batch expiry (chain-price bump)

Goal: drive the cluster to committed depth 3, then **force a radius decrease** by making the
postage batch expire early — done by **raising the chain price** so the batch's fixed balance is
consumed faster — and observe whether the decrease happens and how pull-sync / the redistribution
game behave afterwards. This is the faithful October-style trigger (a network commitment drop
lowers the required radius), unlike node-removal.

## Setup

- Patched bee image (cap 4000 + `threshold=capacity` + wakeup 10s) on a 6-node `local-dns` cluster.
- All 6 full nodes **staked** (1e17 each) via the reserve-radius check's `ensureStaked` step.
- `ci-reserve-radius` drive mode drove the cluster to **committed depth 3 / storage radius 3**
  (`max-committed-depth: 3`, `max-uploads: 0`). One postage batch (`c6a64ec7`, depth 22) on the
  uploader node (bee-1).
- Chain price changed via `storage-incentives`: `npx hardhat changeprice --price <P> --network localhost`.

## Finding 1 — chain price sets batch TTL (inverse-proportional), and the oracle clamps per call

`batchTTL` (seconds remaining on `/stamps`) scales as **~1 / chain-price** — the batch has a fixed
paid balance, and raising the price drains it proportionally faster:

| chain price | batch `batchTTL` | note |
|---|---|---|
| 30000 (baseline) | **86126 s (~24 h)** | `postage-ttl: 24h` at purchase |
| 1,611,392 | **~1590 s (~26 min)** | price ↑ 53.7× → TTL ↓ 54.2× — inverse-proportional ✅ |
| 4,000,000 | **~640 s (~11 min)** | TTL ↓ ~2.5× again; batch then expired on schedule |

**The price oracle clamps the per-update *step*.** `changeprice --price 10000000` from 30000 only
moved it to **1,611,392**; an immediate re-call to 10M was a **no-op**. But a smaller step from
there, `--price 4000000`, *did* apply (→ 4,000,000). So the oracle bounds how far the price can move
per call — you raise it in **steps** (`30000 → 1.6M → 4M → …`), not one big jump. Command:
`cd ~/Git/ethersphere/storage-incentives && npx hardhat changeprice --price <P> --network localhost`.

## Finding 2 — yes, raising the price decreases the radius (3 → 2)

Observed sequence (single batch on bee-1):

| time | event |
|---|---|
| ~22:38 | price at 4M, `batchTTL ~300s` (not expired yet) → **radius already dropped 3 → 2**, `within_radius` jumped **0 → ~2215** |
| 22:43 | batch **EXPIRED** (`batchTTL` → 0) |
| ~22:50 | expired chunks **evicted** → `within_radius` → 0, reserve emptied |

Two takeaways:
- **The decrease is driven by the price rise (commitment drop), not (only) final expiry** — the radius
  fell 3 → 2 while the batch still had ~300s TTL. Raising the chain price shrinks the batch's
  committed value, which lowers the network's required depth → the radius decreases.
- **Decreasing the radius expands the neighbourhood**, so `within_radius` jumped 0 → ~2215 (radius 2
  covers more keyspace than radius 3 → real data is now in-radius — the "merge").
- The drop was **non-uniform**: 4 of 6 nodes reached radius 2; **bee-0 and bee-4 stayed at radius 3**
  (a split/inconsistent end state — they were the heaviest re-syncers).

## Finding 3 — at radius 2: game keeps playing, pull-sync works, but it never fully converges here

- **Pull-sync works** — during the merge the survivors actively re-synced the expanded radius-2
  neighbourhood (`pullsyncRate` 0.8–0.9 on bee-0/bee-4, 0.01–0.33 on the others).
- **Staked nodes keep playing the game** — redistribution **round advanced 49 → 64, none frozen**
  (`isFrozen=false` throughout).
- **But no node reached `isFullySynced=true`** — they stayed mid-catch-up. And because this was a
  **single batch**, full expiry then evicted *all* chunks (`within_radius` → 0, `pullsyncRate` → 0),
  so there was nothing left to converge on. The resync never completed before expiry wiped the data.

## Conclusion

- **Chain-price increase is a controllable, faithful decrease trigger** (commitment drop → radius 3→2),
  more reliable than node-removal — and it works *before* full expiry. TTL ≈ 1/price, oracle steps clamped.
- The decrease reproduces the merge mechanism: neighbourhood expands → pull-sync re-syncs (functional)
  → game stays live (rounds advance, no freeze), but does not reach full sync at this scale.
- **Caveat — a single batch is all-or-nothing**: its full expiry wipes the entire reserve
  (`within_radius` → 0), so you can't hold a stable "radius 2 with data" state. To step the radius down
  one level *and retain data*, **stagger multiple batches' expiry** (different balances / creation times
  / topups, sized ~one depth-level each) so commitment is removed incrementally rather than all at once.
