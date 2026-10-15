---
name: change-storage-price
description: >-
  How to change the Bee storage price (the postage price oracle) on a local
  beekeeper/beelocal cluster, so postage batches start expiring and reserve/radius
  behavior can be exercised. Uses the storage-incentives repo's hardhat `changeprice`
  task (`npx hardhat changeprice --price <N> --network localhost`) against the
  PriceOracle contract, with a `cast` fallback. Use whenever the task involves raising
  or lowering the chain storage price, activating pricing so batches expire, batch
  TTL / expiry on the local chain, the price oracle, or `setPrice`.
---

# Changing the Bee storage price on a local cluster

Goal: set the on-chain storage **price** the local Bee nodes read from the postage
price oracle. The local devchain (`bee-localchain`) boots with `currentPrice = 0`,
which makes every batch's TTL **infinite** (`batchTTL = -1`) — so nothing ever expires.
Raising the price activates pricing and starts the expiry clock; that's the lever for
exercising batch expiry → reserve eviction → commitment/radius changes.

> Values below are for the beelocal devchain (`bee-localchain:0.9.4`). Verify the
> addresses and the branch still exist before relying on them.

## The command

```bash
cd "${STORAGE_INCENTIVES_REPO:-$(git rev-parse --show-toplevel)/../storage-incentives}"
git checkout ci/set-price          # branch that carries tasks/changeprice.ts
npm install                        # first time only
npx hardhat changeprice --price 30000 --network localhost
```

It reads and logs `currentPrice()`, calls `setPrice(<price>)` on the PriceOracle, waits
for the tx, then logs the new price. Bee picks up the new price at
`/chainstate` `currentPrice` within a few blocks.

Override the target oracle with `--contract 0x…` (defaults to the cluster oracle below).

## What the pieces are

- **Task**: `tasks/changeprice.ts` on storage-incentives branch **`ci/set-price`**
  (registered as the `changeprice` hardhat task).
- **`--network localhost`** → `url: http://geth-swap.localhost`, `chainId 12345`
  (from `hardhat.config.ts`). This is the beelocal geth-swap RPC exposed via traefik.
  (`localcluster` = `http://geth-swap:8545`, the in-cluster form; `localhost` is what
  you run from your host.)
- **Signer**: the `localhost` network defines no `accounts`, so hardhat uses geth-swap's
  own remote accounts — i.e. the devchain admin **`0x62CAb2b3B55f341F10348720ca18063cDB779ad5`**
  (private key `4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d` — the
  public ci-stake key, safe to hardcode). It holds the role `setPrice` requires.
- **PriceOracle (default target)**: **`0x538E6dE1D876BBCD5667085257bc92F7c808A0F3`** —
  `setPrice(uint32)` / `currentPrice()`, wired to the active PostageStamp
  (`0x657241f4494A2F15Ba75346E691d753A978C72Df`). Deterministic on
  `bee-localchain:0.9.4`.
  - Note: bee's configured `price-oracle-address 0x5aFE06…` is a *different, unused*
    instance — do **not** target it.

## `cast` fallback (no Node / hardhat)

Same effect, straight to the contract:

```bash
cast send 0x538E6dE1D876BBCD5667085257bc92F7c808A0F3 "setPrice(uint32)" 30000 \
  --private-key 4663c222787e30c1994b59044aa5045377a6e79193a8ead88293926b535c722d \
  --rpc-url http://geth-swap.localhost
```

Read the current price:
```bash
cast call 0x538E6dE1D876BBCD5667085257bc92F7c808A0F3 "currentPrice()(uint256)" \
  --rpc-url http://geth-swap.localhost
```

## Gotchas (measured)

- **uint32 overflow — hard cap 4194303.** `setPrice` does `_price << 10` in uint32, so
  any `_price > 4194303` silently **wraps** (`setPrice(5000000)` → `805696`). Keep the
  price ≤ 4194303; use `4000000` when you want "large".
- **Price 0 ⇒ infinite TTL.** With `currentPrice = 0` every batch (amount 1) has
  `batchTTL = -1` and never expires — dilution/expiry is a no-op. Run `changeprice` once
  (e.g. `--price 10000`) just to *activate* pricing before expecting any expiry.
- **24h minimum validity once priced.** After pricing is active, the PostageStamp
  contract rejects batches whose amount buys < 24h of validity ("insufficient amount for
  24h minimum validity"). So a batch's `postage-ttl` must be ≥ 24h; pushing it to expiry
  then needs many dilution/depth increments.
- **PostageStamp `lastPrice()` starts at 0** — the first `setPrice` is also what
  activates pricing on the stamp contract.
- **Verify it landed** on a Bee node: `curl http://bee-0.localhost/chainstate` and check
  `currentPrice`.

## Reset

Price is chain state — it **persists** across cluster restarts on the same devchain.
To lower it back, run `changeprice` again with the smaller value (or `--price 0` to fully
deactivate pricing / restore infinite TTL).
