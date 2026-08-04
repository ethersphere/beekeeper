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

## 5. CAC, same index, equal timestamp

| Field           | Same / different                                                                 |
| --------------- | -------------------------------------------------------------------------------- |
| Address         | different (two CACs in the same postage bucket)                                  |
| Batch           | same                                                                             |
| Stamp           | same batch / index / timestamp; signatures differ (each stamp is bound to its CAC address) |
| Stamp index     | same                                                                             |
| Stamp timestamp | same                                                                             |

**Expected:** the lexicographically lower CAC address wins; the higher-address CAC is absent.

```bash
# from beekeeper repo, with cluster local-dns up and context k3d-bee
./dist/beekeeper check \
  --config config/beekeeper-local.yaml \
  --config-dir config \
  --cluster-name local-dns \
  --checks ci-chunk-convergence
```
