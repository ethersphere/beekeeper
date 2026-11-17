---
description: Verify the local beelocal + beekeeper cluster is healthy and ready for checks
argument-hint: [cluster-name]
---

Read-only health check of the local k3d substrate and the Bee nodes beekeeper deployed
onto it. Cluster name: "$1" if given, otherwise `local-dns`. Do not modify the cluster.

Take the namespace, api-scheme and api-domain from the cluster's definition in
`config/local.yaml` rather than assuming; for `local-dns` that is namespace `local` and
`http://<node>.localhost` (port 1633) through the ingress, with node groups
`bootnode-0`, `bee-0..N` and `light-0..M`.

Work bottom-up and stop drilling only when a layer hard-fails in a way that blocks the
next one (no k3d cluster ⇒ nothing above can pass).

1. **Tooling** — `kubectl`, `k3d`, `docker`, `curl`, `jq` present, and
   `./dist/beekeeper` exists (note `make binary` if missing; do not build unless asked).

2. **Substrate** — `k3d cluster list` shows all nodes up; the registry
   (`curl -s http://k3d-registry.localhost:5000/v2/_catalog`) lists `ethersphere/bee`;
   the geth-swap pods are Running.

3. **Workloads** — `kubectl get pods -n <ns> -o wide`: bootnode, every `bee-*` and
   `light-*` pod Running and Ready, none in `CrashLoopBackOff`/`Error`/`Pending`. Each
   pod's image is the locally-pushed `k3d-registry.localhost:5000/ethersphere/bee:…` —
   confirm the tag is the one under test. Skim `kubectl logs -n <ns> <pod> --tail=50`
   of one full node for fatal errors.

4. **Ingress** — routes exist (`kubectl get ingress,ingressroute -n <ns>`) and each
   node answers `/health` and `/readiness` through them.

5. **Per-node baseline** — for every full node, via `http://<node>.localhost`:
   - `/addresses` → overlay,
   - `/topology` → `connected`, `population`, `depth` (a small cluster should be a near
     full mesh; `connected > 0` everywhere),
   - `/status` → `storageRadius`, `reserveSize`, `reserveSizeWithinRadius`,
     `pullsyncRate`, `committedDepth`, `isReachable`, `beeMode`,
   - `/reservestate` → `radius`, `storageRadius`, `commitment`.

   A fresh local cluster normally shows `storageRadius: 0` and a small reserve. This
   snapshot is the baseline for any reserve/radius work.

6. **Report** — one PASS/WARN/FAIL line per layer with its single most useful piece of
   evidence, a per-node table
   (`node | overlay | connected | storageRadius | reserveSize | pullsyncRate`), and a
   verdict: ready for checks, or not ready plus the blocking item. If ready, suggest
   the next check, e.g.
   `./dist/beekeeper check --cluster-name=<name> --checks=ci-pingpong --log-verbosity=debug`.

Keep it concise and evidence-driven; quote the actual output behind any FAIL.
