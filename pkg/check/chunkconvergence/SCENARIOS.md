# Chunk convergence check scenarios

Check: `ci-chunk-convergence` (`chunk-convergence`)

**Common setup:** pick the closest full-node pair, upload conflicting chunks to the two nodes, wait for sync, then assert both nodes agree on the expected outcome.

---

## 1. Divergent SOC, identical stamp

| Field           | Same / different                                                   |
| --------------- | ------------------------------------------------------------------ |
| Address         | same                                                               |
| Batch           | same                                                               |
| Stamp           | same (identical stamp on both uploads)                             |
| Stamp index     | same                                                               |
| Stamp timestamp | same                                                               |

**Expected:** the SOC with the lexicographically lower wrapped CAC wins.

---

## 2. SOC, same index, higher timestamp

| Field           | Same / different                                                   |
| --------------- | ------------------------------------------------------------------ |
| Address         | same                                                               |
| Batch           | same                                                               |
| Stamp           | different (same index, different timestamp → different stamp hash) |
| Stamp index     | same                                                               |
| Stamp timestamp | different                                                          |

**Expected:** the higher timestamp wins.

---

## 3. SOC, different index, higher timestamp

| Field           | Same / different                                                   |
| --------------- | ------------------------------------------------------------------ |
| Address         | same                                                               |
| Batch           | same                                                               |
| Stamp           | different                                                          |
| Stamp index     | different (two envelopes)                                          |
| Stamp timestamp | different                                                          |

**Expected:** the higher timestamp wins.

---

## 4. CAC, same index, higher timestamp

| Field           | Same / different                                                   |
| --------------- | ------------------------------------------------------------------ |
| Address         | different (two CACs in the same postage bucket)                    |
| Batch           | same                                                               |
| Stamp           | different (signed over different addresses + different timestamps) |
| Stamp index     | same                                                               |
| Stamp timestamp | different                                                          |

**Expected:** the higher-timestamp CAC is present; the lower-timestamp CAC is absent.

---

## 5. SOC, cross-batch, higher timestamp

| Field           | Same / different                                                   |
| --------------- | ------------------------------------------------------------------ |
| Address         | same                                                               |
| Batch           | different                                                          |
| Stamp           | different                                                          |
| Stamp index     | different (independent envelopes in different batches)             |
| Stamp timestamp | different                                                          |

**Expected:** the higher timestamp wins.

---

## 6. SOC, cross-batch, equal timestamp

| Field           | Same / different                                                   |
| --------------- | ------------------------------------------------------------------ |
| Address         | same                                                               |
| Batch           | different                                                          |
| Stamp           | different                                                          |
| Stamp index     | different (independent envelopes; equality is not forced)          |
| Stamp timestamp | same                                                               |

**Expected:** the stamp with the lexicographically lower stamp hash wins.

**Note:** equal timestamp does not mean the same stamp. The two claims are distinct postage stamps (different batches / stamp hashes) that share only the timestamp value.

---

## 7. SOC, stale stamp redelivery race

> **This branch (`convergence-issue`) runs only this scenario** via `ci-chunk-convergence`.

Parallel race of three divergent SOCs at one address: reused lower-timestamp stamp S (payloads P₁ / P₃) against higher-timestamp stamp T (payload P₂). Wrapped(P₃) is lexicographically lower than wrapped(P₁) and wrapped(P₂).

| Field           | Same / different                                                                 |
| --------------- | -------------------------------------------------------------------------------- |
| Address         | same                                                                             |
| Batch           | different (stamp S on batch A, stamp T on batch B)                               |
| Stamp           | S reused on SOC₃ (identical bytes); T is a distinct higher-ts stamp              |
| Stamp index     | different (independent envelopes per batch)                                      |
| Stamp timestamp | different (ts(S) < ts(T)); SOC₃ keeps ts(S)                                      |
| Payload         | three divergent SOCs; wrapped(P₃) lower than wrapped(P₁) and wrapped(P₂)         |

**Sequence (parallel uploads):** SOC₁ under stamp S → Sam; SOC₂ under stamp T → Sara; SOC₃ under stamp S → both nodes.

**Expected:** after sync, every neighborhood node holds P₃.

**Run (local k3d / beekeeper):**

```bash
# from beekeeper repo, with cluster local-dns up and context k3d-bee
./dist/beekeeper check \
  --config config/beekeeper-local.yaml \
  --config-dir config \
  --cluster-name local-dns \
  --checks ci-chunk-convergence
```
