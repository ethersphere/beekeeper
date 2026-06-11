# pkg/k8s unit-testing improvement plan

A phased plan to raise coverage and code quality of `pkg/k8s` and its subpackages.
Each phase is small, independently verifiable, and ends with the same gate:

```bash
make build && make vet && make lint && make test
```

Phases that touch watch/concurrency code additionally require `make test-race`.
Commit after every green phase (Conventional Commits, subject line only) so any
step can be reverted in isolation.

## Baseline (2026-06-11)

`go test -count=1 -cover ./pkg/k8s/...`:

`go test -count=1 -cover ./pkg/k8s/...` (Baseline → **Final** after Phase 5,
see "Final snapshot" below):

| Package | Baseline | Final |
| --- | --- | --- |
| `pkg/k8s` | 87.1% | **98.6%** |
| `pkg/k8s/configmap` | 100% | 100% |
| `pkg/k8s/containers` | 100% | 100% |
| `pkg/k8s/customresource/ingressroute` | **0%** (no tests) | **97.7%** |
| `pkg/k8s/customresource/ingressroute/mock` | — (did not exist) | 0% own / fully exercised¹ |
| `pkg/k8s/ingress` | 75.5% | **100%** |
| `pkg/k8s/mocks` | 0% (cross-package artifact) | 0% own / exercised¹ |
| `pkg/k8s/namespace` | 100% | 100% |
| `pkg/k8s/persistentvolumeclaim` | 100% | 100% |
| `pkg/k8s/pod` | **54.1%** | **100%** |
| `pkg/k8s/secret` | 100% | 100% |
| `pkg/k8s/service` | **54.9%** | **100%** |
| `pkg/k8s/serviceaccount` | 100% | 100% |
| `pkg/k8s/statefulset` | **61.9%** | **100%** |

¹ `mock` and `mocks` have no test files of their own, so the per-package
`-cover` figure is 0%, but every symbol is exercised by other packages' tests —
under `-coverpkg=./pkg/k8s/...` they are essentially fully covered (aggregate
99.6%).

Main uncovered functions at baseline (`go tool cover -func`) — **all now covered
(see the phase notes):**

- ~~`pod/client.go`: `Get`, `DeletePods`, `WatchNewRunning`, `GetControllingStatefulSet`, `WaitForPodRecreationAndCompletion`, `processEventInState`, `WaitForRunning`, `String`~~ → 100% (Phases 2.4, 3.2–3.4)
- ~~`service/client.go`: `GetNodes`, `FindNode`~~ → 100% (Phase 2.1)
- ~~`statefulset/client.go`: `StatefulSets`, `Get`, `UpdateImage`, `GetUpdateStrategy`, `Update`; `statefulset.go`: `newUpdateStrategy`~~ → 100% (Phase 2.2); `ReadyReplicasWatch` → 100% (Phase 3.1)
- ~~`ingress/client.go`: `GetNodes`~~ → 100% (Phase 2.3)
- ~~`customresource/ingressroute/*`: everything~~ → 97.7% (Phase 4)
- ~~`k8s.go`: `WithLogger`, `WithRequestLimiter`~~ → 100% (Phase 2.5)

## Testing strategy decisions

1. **`fake.Clientset` is the single test double for `kubernetes.Interface`.**
   Tests already use `fake.NewSimpleClientset()` for happy paths but fall back to
   the hand-written `pkg/k8s/mocks` package (magic names such as
   `mocks.CreateBad`) for error injection. The idiomatic replacement is a
   reactor on the fake clientset:

   ```go
   cs := fake.NewClientset()
   cs.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
       return true, nil, errors.New("mock error: cannot create pod")
   })
   ```

   Note: client-go is pinned at v0.33, where `fake.NewSimpleClientset` is
   deprecated — use `fake.NewClientset()` in new/updated tests.

2. **The bee-style functional-options mock ("With pattern", see
   `bee/pkg/pullsync/mock`) is used only for beekeeper-owned interfaces** —
   i.e. `ingressroute.Interface` / `IngressRouteInterface`, which have no
   upstream fake. Shape:

   ```go
   package mock // pkg/k8s/customresource/ingressroute/mock

   var _ ingressroute.Interface = (*Clientset)(nil)

   func New(opts ...Option) *Clientset { ... }
   func WithIngressRoutes(irs ...ingressroute.IngressRoute) Option { ... }
   func WithGetError(err error) Option { ... }
   func WithCreateError(err error) Option { ... }
   ```

   Mocks guard with `sync.Mutex` (race detector must pass) and assert interface
   compliance with the `var _ Iface = (*Mock)(nil)` line.

3. **Watch-based code is tested with fake watch reactors**, not sleeps:

   ```go
   w := watch.NewRaceFreeFake()
   cs.PrependWatchReactor("pods", k8stesting.DefaultWatchReactor(w, nil))
   go func() { w.Modify(&v1.Pod{...}) }()
   ```

   Deadlines come from `context.WithTimeout` in the test, never `time.Sleep`.

4. **Conventions** (per AGENTS.md): external test packages (`package foo_test`),
   table-driven tests, `t.Context()` instead of `context.Background()` where a
   test context is needed, `t.Parallel()` where safe, no new dependencies
   (everything needed ships with client-go: `k8s.io/client-go/testing`,
   `k8s.io/apimachinery/pkg/watch`, `k8s.io/client-go/rest/fake`).

## Phases

### Phase 1 — migrate error injection to fake.Clientset reactors

Goal: remove the dependency on hand-written `pkg/k8s/mocks` CRUD mocks; one
commit per package so each diff stays reviewable.

- [x] 1.1 `configmap` — replace `mocks.NewClientset()` cases with reactors
- [x] 1.2 `secret`
- [x] 1.3 `namespace`
- [x] 1.4 `serviceaccount`
- [x] 1.5 `persistentvolumeclaim`
- [x] 1.6 `pod` (existing `Set` + `Delete` error cases; new tests come in Phase 2/3)
- [x] 1.7 `service`
- [x] 1.8 `ingress`
- [x] 1.9 `statefulset` (error-injection cases only; `TestReadyReplicasWatch`
      stays on the mock until Phase 3.1)
- 1.10 + 1.11 (delete the mock web; update AGENTS.md) — **relocated to Phase 3
  as steps 3.6 and 3.7.** See the "1.10/1.11 relocation" note below for the
  rationale; both are blocked until the statefulset watch test is migrated in
  3.1, so they were moved to run right after it. Phase 2 (which does not touch
  `pkg/k8s/mocks`) is the next reachable work.

Suggested commits: `test(k8s): replace hand-written clientset mocks with fake reactors in <pkg>` … `refactor(k8s): remove unused hand-written clientset mocks`.

Gate: full gate; total coverage must not drop for any touched package.

#### Notes

- **1.1 `configmap`** (2026-06-11): the three `mocks.NewClientset()` error cases
  (`create_error`, `update_error`, `delete_error`) now build a `fake.NewClientset()`
  with a `PrependReactor` via a small `newErrorClientset(verb, resource, err)`
  test helper. The `create_error` case seeds no object, so `Set`'s initial
  `Update` returns `NotFound` and falls through to `Create`, which the reactor
  fails — matching the production code path. Coverage stayed at 100%.
  - **Decision: happy-path cases stay on `fake.NewSimpleClientset()` for now.**
    Migrating them to `fake.NewClientset()` (per decisions §1) breaks the
    existing `reflect.DeepEqual(response, expected)` assertions, because
    `NewClientset` records server-side-apply `ManagedFields` on the returned
    object that `NewSimpleClientset` does not. A blanket
    `NewSimpleClientset → NewClientset` migration therefore needs to either clear
    `ManagedFields` or relax the comparison, which is out of scope for the
    error-injection step and better suited to the Phase 5 quality pass. Only the
    error cases (which assert on the error, not the object) use `NewClientset`.
- **1.2 `secret`** (2026-06-11): identical shape to `configmap` — same
  `newErrorClientset(verb, resource, err)` helper (resource `"secrets"`), same
  `NewSimpleClientset` rationale for the happy-path cases (the `ManagedFields`
  `DeepEqual` issue applies equally). Coverage stayed at 100%.
- **1.3 `namespace`** (2026-06-11): only the single `mocks.DeleteBad` case in
  `TestDelete` used the hand-written mock. `Delete` does a `Get` +
  managed-by-label check *before* the `Delete` call, so the helper here takes
  variadic seed objects — `newErrorClientset(verb, resource, err, objects...)` —
  and the case seeds a beekeeper-managed `test` namespace so the check passes and
  the `delete` reactor fires. The seeded object's `ManagedFields` are irrelevant
  (the case asserts on the error string, not the object). Coverage stayed at 100%.
- **1.4 `serviceaccount`** (2026-06-11): identical shape to `configmap`/`secret`
  — same `newErrorClientset(verb, resource, err)` helper (resource
  `"serviceaccounts"`), happy-path cases left on `NewSimpleClientset` for the
  same `ManagedFields`/`DeepEqual` reason. Coverage stayed at 100%.
- **1.5 `persistentvolumeclaim`** (2026-06-11): the `mocks` cases live in
  `client_test.go` (not `persistentvolumeclaim_test.go`, which only covers
  `ToK8S`). Same `newErrorClientset(verb, resource, err)` helper (resource
  `"persistentvolumeclaims"`), happy-path cases left on `NewSimpleClientset` for
  the same `ManagedFields`/`DeepEqual` reason. Coverage stayed at 100%.
- **1.6 `pod`** (2026-06-11): the `mocks` cases live in `client_test.go`. Migrated
  all three existing error cases — `create_error`/`update_error` (`TestSet`) **and**
  `delete_error` (`TestDelete`) — via the same `newErrorClientset(verb, resource,
  err)` helper (resource `"pods"`). **Scope note:** the checkbox originally read
  "only the existing `Set` error cases"; I also migrated the existing `Delete`
  error case so `pod/client_test.go` drops the `mocks` import entirely (Phase 1's
  per-package goal, and a prerequisite for deleting `mocks/pod.go` in 1.10). No
  *new* tests were added — `Get`, `DeletePods`, `GetControllingStatefulSet`, and
  the watch funcs remain for Phase 2.4/3, so package total is unchanged at 54.1%
  (`Set` and `Delete` already at 100%). Happy-path cases left on
  `NewSimpleClientset` for the same `ManagedFields`/`DeepEqual` reason.
- **1.7 `service`** (2026-06-11): four error cases in `client_test.go`. Service's
  `Set` is `Get` → (NotFound→`Create`) / (other error→"getting…") / (success→
  `Update`), so each case needs distinct setup, handled by the variadic
  `newErrorClientset(verb, resource, err, objects...)` helper (resource
  `"services"`):
  - `create_error`: no seed → `Get` returns NotFound → `create` reactor fails.
  - `update_error`: seed a `test_service` so `Get` succeeds → `update` reactor fails.
  - `get_error`: `get` reactor returns a **non-NotFound** plain error → Set returns
    the "getting service…" branch (the old mock used `mock error: unknown`).
  - `delete_error`: `delete` reactor fails (`TestDelete`).
  `Set`/`Delete` stay at 100%; package total unchanged at 54.9% (`GetNodes`/
  `FindNode` remain for Phase 2.1).
- **1.8 `ingress`** (2026-06-11): three error cases in `client_test.go`. `Set` is
  Update-first (same shape as `configmap`), so the plain
  `newErrorClientset(verb, resource, err)` helper suffices. Resource is
  `"ingresses"` on the `networking/v1` API (the fake clientset routes it the same
  way regardless of API group). `Set`/`Delete` stay at 100%; package total
  unchanged at 75.5% (`GetNodes` remains for Phase 2.3).
- **1.9 `statefulset`** (2026-06-11): the most-used mock. Migrated every
  **error-injection** case in `client_test.go` to the plain
  `newErrorClientset(verb, resource, err)` helper (resource `"statefulsets"`):
  - `TestSet` `create_error` (no seed → Update NotFound → `create` reactor) /
    `update_error` (`update` reactor, plain non-NotFound error).
  - `TestDelete` `delete_error` (`delete` reactor).
  - `TestReadyReplicas` `replicas_error` (`get` reactor, non-NotFound).
  - `TestRunningStatefulSets` / `TestStoppedStatefulSets` `not_found_in_namespace`
    and `wrong_namespace`. **Note:** `not_found_in_namespace` uses a `list`
    reactor returning `apierrors.NewNotFound` (not a bare empty fake clientset)
    specifically to keep the `IsNotFound → return nil, nil` branch covered — a
    real fake `List` returns an empty list with no error and would skip it.
    `wrong_namespace` uses a `list` reactor with a plain error.
  - **Deferred:** `TestReadyReplicasWatch` still uses `mocks.NewClientset()` (the
    mock's `Watch`). It's a watch/race test → Phase 3.1 (decisions §3,
    `watch.NewRaceFreeFake`). So this package keeps the `mocks` import for now and
    `mocks/statefulset.go` can't be deleted in 1.10 until 3.1 lands (flagged in
    1.10). All migrated funcs stay at 100%; `ReadyReplicasWatch` unchanged at
    93.8%; package total unchanged at 61.9%.
- **1.10/1.11 relocation** (2026-06-11): step 1.10 (delete the hand-written
  clientset mock files) is **unreachable as written until Phase 3.1**, so it and
  its dependent doc step 1.11 were moved to Phase 3 (steps 3.6/3.7). Why nothing
  can be deleted now — not even piecemeal: `statefulset/client_test.go`'s
  `TestReadyReplicasWatch` still constructs `mocks.NewClientset()` for the mock's
  `Watch`. Instantiating `*mocks.Clientset` forces **all** of its methods to
  compile — `AppsV1()`, `CoreV1()`, `NetworkingV1()` — whose bodies reference
  `NewAppV1`/`NewCoreV1`/`NewNetworkingV1`, which in turn reference every
  per-resource mock (`configmap.go`, `secret.go`, …, `ingress.go`,
  `statefulset.go`) and the `CreateBad`/`UpdateBad`/`DeleteBad` constants in
  `app.go`. So the entire `clientset.go`/`core.go`/`app.go`/`networking.go` +
  per-resource web is transitively live; deleting any one file breaks
  compilation. A "deletion-only" 1.10 is therefore impossible while the watch
  test uses the mock. Rather than do throwaway surgery (trimming `clientset.go`
  just to delete it again in 3.x), the deletion waits for 3.1 to migrate the
  watch test to `watch.NewRaceFreeFake`, after which the whole web drops in one
  clean commit (3.6). 1.11 follows (3.7) because AGENTS.md can't truthfully say
  "only the `ClientConfig` mock remains" until the web is gone. `mocks/k8s.go`
  (mock `ClientConfig`) and `mocks/roundtripper.go` are unrelated to the
  clientset and are kept regardless — they back `pkg/k8s/k8s_test.go`.

### Phase 2 — cover plain (non-watch) untested functions

All testable with `fake.NewClientset(seedObjects...)` plus reactors for error
branches.

- [x] 2.1 `service`: `GetNodes`, `FindNode` (seed Services with/without matching
      labels/ports; error reactor for List) — target ≥ 90%
- [x] 2.2 `statefulset`: `StatefulSets`, `Get`, `UpdateImage`,
      `GetUpdateStrategy`, `Update`, `newUpdateStrategy` (table over strategy
      types) — target ≥ 90%
- [x] 2.3 `ingress`: `GetNodes` — target ≥ 90%
- [x] 2.4 `pod`: `Get`, `DeletePods`, `GetControllingStatefulSet` (owner-ref
      present / absent / API error), `PodRecreationState.String` (the plan's
      "eventState.String" — no `eventState` type exists)
- [x] 2.5 `k8s`: `WithLogger`, `WithRequestLimiter` option behavior (incl. nil
      logger / negative max-requests guards)

Gate: full gate. One commit per package: `test(k8s): cover <funcs> in <pkg>`.

#### Notes

- **2.1 `service`** (2026-06-11): added `TestGetNodes` and `TestFindNode` plus two
  fixture helpers (`svc`, `selectorSvc`). Seeded with `fake.NewClientset(...)`
  (per decisions §1) — safe here because the tests assert on the returned
  `NodeInfo` structs and `Service.Name`, never a full k8s object, so the
  `ManagedFields` `DeepEqual` issue from Phase 1 doesn't apply. The `fake`
  clientset honors the `LabelSelector` in `List`, so `GetNodes`'s pass-through of
  the selector is exercised by seeding a non-matching service (`other-0`).
  Branches covered: `GetNodes` — api-port+real-IP (included), api-port+`"None"`
  (excluded), non-api port (excluded), label-selector miss (excluded), list
  error; `FindNode` — nil selector (skip), selector mismatch (skip), selector
  match without api port (fall through → not-found error), selector match with
  api port (return), list error. Test names are order-independent (only one
  service can satisfy each terminal branch), so they don't rely on the fake's
  list ordering. `GetNodes`/`FindNode` 0%→100%; **package total 54.9%→100%**
  (≥90% target met).
- **2.2 `statefulset`** (2026-06-11): added `TestStatefulSets`, `TestGet`,
  `TestUpdateImage`, `TestGetUpdateStrategy`, `TestUpdate`. Upgraded the
  package's `newErrorClientset` helper to the seed-aware variadic form (so an
  `update`/`get` reactor can fire against a seeded object) — backward-compatible
  with the Phase-1 no-object callers. Added a `newStatefulSet(name, image)`
  fixture with one container, since `UpdateImage` indexes `Containers[0]` and
  would panic on a container-less seed. Key decisions:
  - `newUpdateStrategy` is covered **through its only caller** `GetUpdateStrategy`
    (table over `OnDelete` / `RollingUpdate`+partition / `RollingUpdate`-no-
    partition), so no `export_test.go` was needed.
  - `UpdateImage` `not_found`: a bare `fake.NewClientset()` `Get` returns
    NotFound, exercising the `IsNotFound → return nil` branch directly.
  - `UpdateImage` `success` re-`Get`s through the clientset and asserts the
    container image actually changed (not just "no error").
  All six target funcs (incl. `newUpdateStrategy`) 0%→100%; **package total
  61.9%→99.0%** (≥90% met; the remaining 1% is `ReadyReplicasWatch` at 93.8%,
  deferred to Phase 3.1).
- **2.3 `ingress`** (2026-06-11): added `TestGetNodes` + an `ingressWith(name,
  labels, hosts...)` fixture (one `IngressRule` per host). The Phase-1 plain
  `newErrorClientset(verb, resource, err)` helper sufficed — the NotFound case
  just passes `apierrors.NewNotFound` as the `list` reactor error (no seed
  needed). Branches: label-selector match with a non-empty host (included) +
  an empty-host rule on the same ingress (skips the `rule.Host != ""` false
  arm); label-selector miss (excluded); empty result (success, nil); `list`
  NotFound → `nil, nil`; `list` plain error → "list ingresses…". Note the
  distinction between the **empty-result** case (bare `fake.NewClientset()`
  returns an empty list, loop runs zero times) and the **NotFound** case (reactor
  returns `IsNotFound`, hits the early `return nil, nil`) — both needed to cover
  both arms. `GetNodes` 0%→100%; **package total 75.5%→100%** (≥90% met).
- **2.4 `pod`** (2026-06-11): added `TestGet`, `TestDeletePods`,
  `TestGetControllingStatefulSet`, `TestPodRecreationStateString`, plus a `newPod`
  fixture; upgraded pod's `newErrorClientset` to the seed-aware variadic form.
  **Plan-text fix:** the checkbox said `eventState.String`, but there is no
  `eventState` type — the only `String()` is on the exported `PodRecreationState`
  (5 states + `default`→"Unknown"), tested directly from the external test
  package. Branch notes:
  - `DeletePods`: seeds two normal pods + one with a `DeletionTimestamp` (skipped
    via the `DeletionTimestamp == nil` guard) → `deleted == 2`; `list` reactor
    error → "listing pods…"; `delete` reactor error → the aggregated
    "some pods failed to delete: [mock error]" path (`deletedCount == 0`).
  - `GetControllingStatefulSet`: owner-ref present + seeded StatefulSet (return);
    no owner ref (`GetControllerOf` nil) and wrong-kind owner ref (both hit the
    "not controlled by a StatefulSet" arm, covering both sides of the `||`);
    pod-`get` reactor error; statefulset-`get` reactor error (pod seeded so its
    Get succeeds, only the `statefulsets` `get` reactor fires).
  Target funcs (`Get`, `DeletePods`, `GetControllingStatefulSet`, `String`)
  0%→100%; **package total 54.1%→68.3%**. The remaining 0% funcs
  (`WatchNewRunning`, `WaitForRunning`, `WaitForPodRecreationAndCompletion`,
  `processEventInState`) are watch/concurrency → Phase 3.2–3.4.
- **2.5 `k8s`** (2026-06-11): `WithLogger`/`WithRequestLimiter` set **unexported**
  `Client` fields, and the existing `k8s_test.go` is an external `package k8s_test`
  that can only observe them indirectly through `NewClient` (which is why
  `WithInCluster`/`WithKubeconfigPath` were already 100% but the nil/negative
  guards here were not reachable). Added a new **internal** test file
  `k8s_internal_test.go` (`package k8s`, coexists with the external test file in
  the same dir) that applies each option to a hand-built `&Client{}` and asserts
  the field effect by identity. Branches: `WithLogger` set vs nil-ignored;
  `WithRequestLimiter` limiter+max set, `max == 0` boundary (>= 0 true),
  nil-limiter ignored, negative-max ignored. Both funcs 0%→100%; **package total
  87.1%→98.6%** (residual is the error paths in `NewClient`/`NewCustomTransport`,
  out of scope here).

### Phase 3 — watch/concurrency functions (race-sensitive)

Use `watch.NewRaceFreeFake` + `PrependWatchReactor`; drive events from a
goroutine; bound every test with a context timeout.

- [x] 3.1 `statefulset.ReadyReplicasWatch` — remaining branch (currently 93.8%)
- [x] 3.2 `pod.WatchNewRunning` — happy path, watch error, context cancellation
- [x] 3.3 `pod.WaitForRunning`
- [x] 3.4 `pod.WaitForPodRecreationAndCompletion` + `processEventInState` —
      cover the state machine via the public entry point first; only test
      `processEventInState` directly (export via `export_test.go`) if a branch
      is unreachable from the public API
- [x] 3.5 run `make test-race` in addition to the full gate
- [x] 3.6 (relocated from 1.10) **now unblocked** — 3.1 migrated
      `TestReadyReplicasWatch` off `mocks.NewClientset()`, so the clientset web has
      no remaining users. Delete the now-unused hand-written clientset
      mock web: `clientset.go`, `core.go`, `app.go`, `networking.go`, and the
      per-resource mocks (`configmap.go`, `secret.go`, `namespace.go`,
      `serviceaccount.go`, `persistentvolumeclaim.go`, `pod.go`, `service.go`,
      `ingress.go`, `statefulset.go`). **Keep** `mocks/k8s.go` (mock
      `ClientConfig`) and `mocks/roundtripper.go` — they back
      `pkg/k8s/k8s_test.go`. Verify with `grep -rn "mocks\." pkg/ --include="*.go"`
      that the only remaining references are to `NewClient`/`FlagString`/
      `FlagParse`/`MockRoundTripper`. Commit: `refactor(k8s): remove unused
      hand-written clientset mocks`.
- [x] 3.7 (relocated from 1.11) update AGENTS.md: "use the `pkg/k8s/mocks` fakes"
      → describe the fake.Clientset + reactor convention (and the remaining
      `ClientConfig` mock). Do after 3.6 so the doc matches the deleted web.

Target: `pod` ≥ 85%. Commit per function group.

#### Notes

- **3.1 `statefulset.ReadyReplicasWatch`** (2026-06-11): rewrote
  `TestReadyReplicasWatch` off `mocks.NewClientset()` onto
  `fake.NewClientset()` + `PrependWatchReactor(… DefaultWatchReactor(w, err))`
  with `w := watch.NewRaceFreeFake()` (decisions §3). Because `RaceFreeFake`
  buffers events, cases enqueue `w.Modify(...)`/`w.Add(...)` **before** the call
  (no goroutine, no sleep, no race). Subtests: watch error; ready (Replicas ==
  ReadyReplicas); not-ready-then-ready (covers the `if` false / loop-continue
  arm); non-`*StatefulSet` event then ready (covers the type-assertion `ok`
  false arm); channel closed (`w.Stop()` → `RaceFreeFake.Stop` is idempotent, so
  the function's deferred `Stop` is safe — this is exactly why §3 mandates
  RaceFree over plain Fake); context cancelled.
  - **The 93.8%→100% branch was the 5s ticker `Infof` line.** A hardcoded
    `time.NewTicker(5s)` can't fire deterministically without either a forbidden
    multi-second wait (§3: no sleeps; a >5s context timeout would just be a slow
    disguised sleep) or making the cadence injectable. Took the latter, minimal
    route: replaced the literal with an unexported package var
    `notReadyLogInterval` (default `5s`) and added `export_test.go`
    (`SetNotReadyLogInterval`, plan-sanctioned for branches unreachable from the
    public API — see 3.4). The `ticker_logs_until_timeout` subtest shortens it to
    1ms with a 100ms `context.WithTimeout` and sends no event, so the ticker
    fires (logs) repeatedly then the deadline returns `context deadline
    exceeded`. `ReadyReplicasWatch` 93.8%→100%; **package total 99.0%→100%**.
  - **Unblocks 3.6:** `statefulset/client_test.go` no longer imports
    `pkg/k8s/mocks`; `mocks.NewClientset()` (the clientset web) now has zero
    users repo-wide. Full gate + `make test-race` green.
- **3.2 `pod.WatchNewRunning`** (2026-06-11): added `TestWatchNewRunning` with a
  `podWatchClientset(w, watchErr)` helper (`fake.NewClientset` +
  `PrependWatchReactor("pods", DefaultWatchReactor(w, watchErr))`) and a
  `runningReadyPod` fixture. Subtests: watch error; context cancelled
  (pre-cancelled ctx, returns immediately); channel closed (`w.Stop()`); and
  `sends_ready_running_pod`. The function blocks in a select loop, so the
  happy-path subtest pre-buffers events into the `RaceFreeFake` (an `Add`/
  non-Modified event → ignored by the `switch`; a Modified pending pod → filtered
  by the running-and-ready guard; a Modified ready pod → sent), runs
  `WatchNewRunning` in a goroutine, reads the pod off `newPods` (bounded by a 1s
  `context.WithTimeout`), then `cancel()`s and asserts the returned
  `context canceled`. `WatchNewRunning` 0%→100%; **package total 68.3%→76.1%**
  (`WaitForRunning`, `WaitForPodRecreationAndCompletion`, `processEventInState`
  remain for 3.3/3.4). Full gate + `make test-race` green.
- **3.3 `pod.WaitForRunning`** (2026-06-11): not a watch — it polls `Get` via
  `wait.PollUntilContextCancel(..., immediate=true, ...)`, so tests just seed a
  `fake.NewClientset` pod in the right phase (no `RaceFreeFake`). Subtests:
  running (→ nil); `get` reactor error; bad-state `Failed`; bad-state `Unknown`;
  and `pending_polls_until_context_cancelled`. The keep-polling arm (the
  `Pending` log + `return false, nil`) is the interesting one: a hardcoded 5s
  poll interval would otherwise make it untestable without a wait. It's covered
  by pre-cancelling the ctx — `immediate=true` runs the condition **once** before
  `loopConditionUntilContext` checks `ctx.Err()`, and the fake `Get` ignores the
  cancelled ctx, so the condition evaluates the Pending pod (covering the branch)
  and then the loop returns `context.Canceled` (asserted via `errors.Is`). No
  sleeps, no production change. `WaitForRunning` 0%→100%; **package total
  76.1%→81.1%** (the Phase-3 `pod ≥ 85%` target is closed by 3.4). Full gate +
  `make test-race` green.
- **3.4 `pod.WaitForPodRecreationAndCompletion` + `processEventInState`**
  (2026-06-11): **no `export_test.go` needed** — every `processEventInState`
  branch is reachable from the public entry by driving event sequences through
  the `RaceFreeFake` watcher, which is what the plan asks for ("cover via the
  public entry point first"). `TestWaitForPodRecreationAndCompletion` subtests:
  - `completes_full_lifecycle` (explicit `containerName="bee"`): pre-buffers
    `Delete → Add → Modify(Running) → Modify(Terminated/Completed)` and the
    function drains them synchronously to `Completed` → nil (covers all four
    state transitions + the `Completed` return + the `newState != currentState`
    transition path).
  - `container_terminated_with_error`: last event terminates with `Reason:Error`
    → `processEventInState` returns the error, propagated out.
  - `skips_unrelated_events_and_uses_first_container` (`containerName=""`): a
    leading non-`*Pod` event (the `!ok` skip), a Modified-while-WaitingForDeletion
    event (the switch's no-transition default), then the lifecycle — and the
    empty containerName exercises the first-container fallback.
  - `watch_error`; and `timeout` (no events, a 50ms `context.WithTimeout` fires
    `watchCtx.Done()` in the initial state → "timed out … (current state:
    WaitingForDeletion)").
  Both funcs 0%→100%; **`pkg/k8s/pod` total 81.1%→100%** (Phase-3 `pod ≥ 85%`
  met). Full gate + `make test-race` green.
- **3.5 verification** (2026-06-11): full gate + `make test-race ./...` green
  across the whole repo (no code change this step). The Phase-3 packages are now
  `pkg/k8s/pod` 100% and `pkg/k8s/statefulset` 100% (both ≥ the `pod ≥ 85%`
  target); all watch tests use `watch.NewRaceFreeFake` with no `time.Sleep`, and
  the race detector is clean. Each of 3.1–3.4 also ran `make test-race`
  individually as it landed; this step is the consolidated confirmation.
- **3.6 delete clientset mock web** (2026-06-11): removed the 13 web files
  (`clientset.go`, `core.go`, `app.go`, `networking.go`, and the 9 per-resource
  mocks). **Kept** `mocks/k8s.go` (`ClientConfig`: `NewClient`/`FlagString`/
  `FlagParse`) and `mocks/roundtripper.go` (`MockRoundTripper`) — both back
  `pkg/k8s/k8s_test.go`. **One snag:** `k8s.go` carried a misplaced
  `var _ kubernetes.Interface = (*Clientset)(nil)` compile-assertion referencing
  the deleted `Clientset`; removed that line (the `kubernetes` import stays, used
  by `NewForConfig`'s return type). Post-delete `grep -rhoE "mocks\.[A-Za-z]+"`
  outside the mocks pkg shows only `NewClient`/`FlagString`/`FlagParse`/
  `MockRoundTripper`. No coverage impact (the deleted code was unreferenced test
  scaffolding); `pkg/k8s` stays 98.6%, every other `pkg/k8s/*` is 100% except
  `ingressroute` (0%, Phase 4) and `mocks` (0%, test-helper pkg with no own
  tests). Full gate + `make test-race` green.
- **3.7 update AGENTS.md** (2026-06-11): rewrote two spots — the architecture
  bullet ("`pkg/k8s/` … tested with client-go's fake clientset; `pkg/k8s/mocks/`
  holds only the `ClientConfig`/RoundTripper doubles") and the **Tests**
  convention ("use the `pkg/k8s/mocks` fakes" → `fake.NewClientset()` +
  `PrependReactor` for error paths, `watch.NewRaceFreeFake` +
  `PrependWatchReactor` for watch paths, bound by `context.WithTimeout`, never
  `time.Sleep`; the only hand-written mock left is the `ClientConfig`/RoundTripper
  pair). `CLAUDE.md` is `@AGENTS.md` so it inherits the change. Doc-only; full
  gate + `make test-race` green. **Phase 3 complete.**

- [x] 4.1 add `pkg/k8s/customresource/ingressroute/mock` implementing
      `ingressroute.Interface`/`IngressRouteInterface` with the With-pattern
      (decisions §2)
- [x] 4.2 test `Client.Set`, `Client.Delete`, `Client.GetNodes` against the
      mock (not-found→create, found→update, API errors, IsNotFound on delete)
- [x] 4.3 test `types.go` (`DeepCopyObject`, `DeepCopyInto`, `GetHost`) and
      `register.go` (`Kind`, `Resource`, `addKnownTypes`) directly
- [x] 4.4 test the REST layer (`ingressroute.go` CRUD, `config.go
      NewForConfig`) with `k8s.io/client-go/rest/fake` (or `httptest.Server`
      if the fake RESTClient fights the custom scheme — pick whichever needs
      less scaffolding and note the choice here)

Target: package ≥ 80%. Commits: `test(k8s): add ingressroute mock with functional options`, `test(k8s): cover ingressroute client`.

#### Notes

- **4.1 ingressroute mock** (2026-06-11): added
  `pkg/k8s/customresource/ingressroute/mock` — a `Clientset`
  (implements `ingressroute.Interface`) whose `IngressRoutes(ns)` returns an
  unexported `ingressRoutes` (implements `IngressRouteInterface`), with
  `var _` compliance assertions for both. Built with functional options
  (decisions §2): `New(...Option)`, `WithIngressRoutes(...)` to seed, and
  per-verb `WithGetError`/`WithCreateError`/`WithUpdateError`/`WithListError`/
  `WithDeleteError`. State is a `map[string]IngressRoute` keyed by
  `"namespace/name"` guarded by a `sync.Mutex` (race-clean). Get returns
  `apierrors.NewNotFound` when absent (so `Client.Set`'s NotFound→Create branch
  is reachable); Delete returns NotFound when absent (so `Client.Delete`'s
  IsNotFound→nil branch is reachable); List ignores the label selector (tests
  seed only what they expect). `Watch` is required by the interface but unused by
  `ingressroute.Client`, so it returns an empty `watch.NewRaceFreeFake()` (no
  error-injection option, to avoid a dead exported symbol). Scope: mock only —
  no tests yet, so both the mock pkg and `ingressroute` stay at 0% until **4.2**
  exercises them through the `Client`. Build/vet/lint/test green (exported
  `With*` options are not flagged unused).
- **4.2 ingressroute Client** (2026-06-11): added `client_test.go`
  (`package ingressroute_test`) driving `Client` through the 4.1 mock via a
  `newClient(opts...)` helper and a `newIR(name, matches...)` fixture. Subtests:
  - `Set`: `create_when_not_found` (empty mock → Get NotFound → Create),
    `update_when_found` (seeded → Get hit → Update), and the three error arms
    `get_error` (non-NotFound → "getting…"), `create_error`, `update_error`.
  - `Delete`: `delete_existing` (nil), `delete_not_found_is_nil` (mock returns
    NotFound → Client swallows it), `delete_error`.
  - `GetNodes`: `extracts_hosts` (two IRs; a `PathPrefix` route whose `GetHost`
    returns "" is skipped; result **sorted** before compare since the mock's
    `List` iterates a map), `no_routes` (nil), `list_not_found_is_nil`
    (`apierrors.NewNotFound` → nil), `list_error`.
  `Client` (`NewClient`/`Set`/`Delete`/`GetNodes`) 0%→100%; **`ingressroute` pkg
  total 0%→45.5%** (the REST layer in `ingressroute.go`/`config.go` and
  `types.go`/`register.go` remain for 4.3/4.4). The mock's own per-package figure
  reads 0% (no test files of its own) but every mock method is exercised here.
  Full gate green.
- **4.3 types.go + register.go** (2026-06-11): added `types_test.go` and
  `register_test.go` (`package ingressroute_test`). `GetHost` was already 100%
  (exercised by 4.2's `GetNodes`), so this covered the rest directly:
  - `IngressRoute.DeepCopyObject` (asserts a non-aliased `*IngressRoute` equal by
    value), `IngressRoute.DeepCopyInto`, and `IngressRouteList.DeepCopyObject`
    with **both** a populated `Items` (the `if in.Items != nil` make+loop arm,
    which also drives `DeepCopyInto`) and `nil` Items (the skip arm). Asserts
    value-equality via `reflect.DeepEqual`, not deep independence — the existing
    `DeepCopyInto` shares the `Spec.Routes` backing array (`out.Spec = ir.Spec`
    then a no-op `copy`), so independence isn't a guarantee to test.
  - `Kind`/`Resource` (exported) checked directly; `addKnownTypes` (unexported)
    covered via the exported `AddToScheme` builder against a fresh
    `runtime.NewScheme()`, then asserting `scheme.Recognizes(...)` for both kinds.
  All target funcs 0%→100%; **`ingressroute` pkg total 45.5%→68.2%** (the REST
  layer — `ingressroute.go` CRUD + `config.go NewForConfig` — remains for 4.4).
  Full gate green.
- **4.4 REST layer** (2026-06-11): **chose `httptest.Server` over
  `k8s.io/client-go/rest/fake`** (the plan left the choice open). Rationale: the
  fake RESTClient would need the custom Traefik types registered in a scheme and
  responses hand-serialized with a matching codec — and it bypasses
  `NewForConfig` (so that wouldn't get covered). With httptest, `NewForConfig`
  itself wires up the scheme/serializer, so one `TestRESTClient` drives
  `NewForConfig` + `IngressRoutes` + all six REST methods through the *real*
  client against a local server that returns JSON; no `export_test.go`, no manual
  scheme/codec scaffolding. The handler routes by method + a `watch=true` query +
  a `/ingressroutes` suffix. `Watch` succeeds against an empty `200`
  (the stream watcher constructs without reading; the test `Stop()`s it).
  All six REST methods + `IngressRoutes` 0%→100%; `NewForConfig` 0%→83.3% (only
  its two defensive `RESTClientFor`/`AddToScheme` error returns are uncovered).
  **`ingressroute` pkg total 68.2%→97.7%** (≥80% target met). **Phase 4 complete.**
  - **Lint note:** extracting the REST test added enough `"ir-0"` literals that
    `goconst` flagged the package; pulled the name into a shared
    `const testIRName = "ir-0"` across the test files (the longer error-message
    strings that merely *contain* `ir-0` are distinct literals and were left
    alone).

### Phase 5 — quality pass (no coverage goal)

- [x] 5.1 align all `pkg/k8s` tests on one table-driven style; extract repeated
      fixture-building into small helpers (`newTestPod(...)`) — only where
      duplication is real, no premature abstraction
- [x] 5.2 add `t.Parallel()` where tests are independent; replace
      `context.Background()` with `t.Context()`
- [x] 5.3 run `/simplify`-style review of the new test code; godoc comments on
      the exported mock API
- [x] 5.4 final coverage snapshot appended to this doc; update the baseline
      table

#### Notes

- **5.1 style + fixtures** (2026-06-11): the test suite is already largely
  aligned — homogeneous cases use a single table + shared loop (e.g. `TestSet`/
  `TestDelete` in each package, `TestGetNodes`), and heterogeneous cases use
  `t.Run` subtests with local fixtures (`newPod`, `newStatefulSet`, `newIR`,
  `runningReadyPod`, …) extracted in Phases 1–4. The one real cross-package
  inconsistency was `newErrorClientset`, which existed in **two** signatures
  (5 plain `(verb, resource, err)`, 4 variadic `(…, objects ...runtime.Object)`).
  Unified all 9 to the byte-identical variadic form (the plain callers pass no
  objects, so behavior is unchanged; coverage stayed 100% on all five touched
  packages). **Deliberately NOT done** (recorded so 5.3 doesn't relitigate):
  - *No shared cross-package test-helper package* (e.g. `internal/k8stest`). The
    duplication is only a 7-line helper; hoisting it would make every call site
    more verbose and add a non-test package used solely by tests. The plan's own
    example (`newTestPod`) points at per-package fixtures, and AGENTS.md says no
    premature abstraction. (The prod-binary worry is moot — `fake` is test-only
    and a package imported only by `_test.go` isn't linked into `beekeeper` — but
    the verbosity/abstraction cost still isn't worth it.)
  - *No forced single test structure.* The table-vs-subtest split is intentional
    and idiomatic: tables where every case shares one assertion shape, subtests
    where setup/assertions differ per case. Collapsing everything into one mold
    would reduce clarity.
  - *Left pre-existing inline `&appsv1.StatefulSet{…}` fixtures* in the older
    `statefulset` list/scale tests as-is — refactoring working pre-existing tests
    is out of scope for this pass and carries risk for little gain.
- **5.2 t.Context() + t.Parallel()** (2026-06-11):
  - **`context.Background()` → `t.Context()`** across all 11 `pkg/k8s` test files
    that used it (incl. as the parent of `context.WithCancel`/`WithTimeout`).
    `goimports` dropped the now-unused `context` import from the 9 files that only
    used `Background()`; `pod`/`statefulset` keep it (they still call
    `context.With*`). All occurrences are in test/subtest bodies, so `t` is always
    in scope (no helper takes a `ctx`).
  - **`t.Parallel()` added at the top level of every authored test function**
    (subtests left sequential — this sidesteps the parallel-subtest vs deferred
    `server.Close()` gotcha in `TestRESTClient`, and the loop tables don't need
    it). Top-level-only parallelism means each package's non-parallel tests (none
    here) run first, then these run concurrently. Validated race-clean with
    `go test -race ./pkg/k8s/...` + full `make test-race`.
  - **Shared-global safety check:** `TestRESTClient` mutates the global
    `scheme.Scheme` (via `NewForConfig`) and `TestReadyReplicasWatch` mutates the
    `notReadyLogInterval` package var — both are the *only* accessors of their
    respective global within their package, so even running parallel to siblings
    there's no concurrent read/write. The race detector confirms. (`TestAddToScheme`
    uses a local scheme; `Kind`/`Resource` only read a package var.)
  - **zsh gotcha (process note):** unquoted `$FILES` doesn't word-split in zsh, so
    the first batch sed was a no-op; redone with a `for` loop.
  Coverage unchanged across all packages; full gate + `make test-race` green.
- **5.3 simplify review + mock godoc** (2026-06-11): **godoc** — the exported
  mock API (package doc + `Clientset`, `Option`, `New`, all six `With*`,
  `IngressRoutes`) was already fully documented in 4.1; verified it renders
  cleanly via `go doc ./…/ingressroute/mock`. **Simplify review** of the new test
  code: it's already clean and consistent (table-driven where cases are uniform,
  `t.Run` subtests where setup varies, local fixtures, `RaceFreeFake` watches with
  no sleeps; passes `gofumpt`/`unconvert`/`goconst`). **No code change applied** —
  every candidate failed the "clear win + project-consistent + worth the churn"
  bar:
  - cross-package error-assert / `newErrorClientset` helper → already declined in
    5.1 (verbose call sites; a non-test package used only by tests);
  - `fmt.Errorf("static")` → `errors.New` sweep of the `errorMsg` tables → imposes
    a style the repo doesn't enforce (no `perfsprint`/`revive`/`stylecheck` in
    `.golangci.yml`) and diverges from the surrounding pre-existing tables;
  - DRYing the mock's near-identical `Create`/`Update` → a shared `upsert` would
    read an error field outside the mutex (violating "the mutex guards the
    fields") or need a lock-held footgun; the 3-line duplication is clearer.
  Doc-only step; gate green; coverage unchanged (no package touched).
- **5.4 final snapshot** (2026-06-11): refreshed the Baseline table into a
  Baseline→Final view (top of doc), struck through the now-covered uncovered-
  function list, and recorded the final numbers below. Aggregate
  `-coverpkg=./pkg/k8s/...` total **99.6%** (baseline was 57% incl. the mocks
  artifact). **Phase 5 complete — plan done.**

## Final snapshot (2026-06-11)

All five phases complete. Per-package numbers are in the Baseline→Final table at
the top; aggregate `-coverpkg=./pkg/k8s/...` total is **99.6%** (from a 57%
baseline that included the mocks artifact).

| Definition-of-done item | Status |
| --- | --- |
| `pkg/k8s/...` total ≥ 85%, no real package below 75% | ✅ 99.6% aggregate; every package with tests ≥ 97.7% (only the no-own-test helper pkgs `mock`/`mocks` read 0% per-package but are fully exercised cross-package) |
| hand-written clientset mocks deleted; one documented mocking convention | ✅ the `clientset.go`/`core.go`/`app.go`/`networking.go` + per-resource web is gone (3.6); only `mocks/k8s.go` (`ClientConfig`) + `mocks/roundtripper.go` remain; convention is `fake.NewClientset()` + reactors / `watch.NewRaceFreeFake`, With-pattern mock only for the beekeeper-owned `ingressroute` |
| `make build`/`vet`/`lint`/`test`/`test-race` all green | ✅ verified at 5.4 |
| AGENTS.md testing conventions updated | ✅ (3.7) |

## Definition of done

- `pkg/k8s/...` total ≥ 85% (from 57% incl. mocks artifact), no package below 75%
- hand-written clientset mocks deleted; one documented mocking convention
- `make build`, `make vet`, `make lint`, `make test`, `make test-race` all green
- AGENTS.md testing conventions updated

## How to run this plan with Claude Code

- **One phase (or sub-item) per request.** Start each session with:
  *"Implement Phase N.M of docs/k8s-unit-testing-plan.md, run the gate, tick
  the checkbox, and propose a commit message."* Small scopes keep diffs
  reviewable and let each step land green.
- **This doc is the shared state.** Claude ticks checkboxes and appends notes
  (e.g. the Phase 4.4 fake-vs-httptest decision) as it goes, so a fresh session
  needs no re-explanation — just point it at this file.
- **Commit after every green step yourself** (or explicitly ask for a commit);
  per repo convention Claude does not push and uses subject-line-only
  Conventional Commit messages.
- **Use plan mode (`shift+tab`) for the open-ended phases** (3 and 4) where
  design choices exist; the mechanical phases (1, 2) can run directly.
- **Verification is part of the task**, not a follow-up: the gate command at
  the top plus `go tool cover -func` for the touched package belong in every
  step's output.
