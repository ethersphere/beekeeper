# AGENTS.md

This file provides guidance to AI coding assistants (Claude Code, Cursor, and others) when working with code in this repository.

## Project overview

Beekeeper is the orchestration and integration-testing tool for [Ethereum Swarm Bee](https://github.com/ethersphere/bee) clusters. A single Go CLI (`cmd/beekeeper`, built on Cobra/Viper) covers several distinct jobs:

- **Cluster lifecycle** — create/delete Bee clusters as Kubernetes resources (`create`, `delete`).
- **Integration checks** — run tests such as `pingpong`, `pushsync`, `retrieval` against a cluster (`check`). This is the primary purpose; checks are also what Bee's CI runs against PRs.
- **Operational tooling against already-running clusters** — `stamper` (postage batch create/topup/dilute/set), `node-funder`/`node-operator` (top up ETH/BZZ), `nuke` (DB reset + resync), `restart`.

Beekeeper consumes Bee as a library (`github.com/ethersphere/bee/v2` in `go.mod`) for shared types (e.g. `swarm.Address`, postage) and talks to running nodes over Bee's HTTP API.

## Essential commands

```bash
make binary   # build ./dist/beekeeper (CGO disabled, version stamped via -ldflags)
make test     # unit tests: go test -v ./pkg/...
make lint     # golangci-lint (pinned v2.10.1; auto-installs if missing)
```

Run a single test: `go test -run TestName ./pkg/<package>/...` (add `-race` to mirror CI).

CI (`.github/workflows/go.yml`) runs `make vet`, `make check-whitespace`, golangci-lint, `make build`, and `make test`. PR titles are linted as Conventional Commits (`.github/workflows/pr-title.yml`).

Requires Go 1.26. For a full local cluster (K3s/k3d + Geth), follow the **Local Development** quick start in `README.md`.

## How a run is wired together

The root command (`cmd/beekeeper/cmd/cmd.go`) builds shared dependencies in `PersistentPreRunE` before any subcommand runs:

1. Loads global config (Viper: `$HOME/.beekeeper.yaml`, flags, env).
2. Loads the **config directory** (cluster/check definitions) from a local path *or* a Git repo.
3. Constructs the logger (optional Loki), the Kubernetes client (unless `--enable-k8s=false`), and the swap/blockchain client (`swap.NotSet{}` when no `geth-url`).

Subcommands receive these via the `command` struct and pass them into runners/clients as constructor args — there is no global singleton.

## Architecture

The big-picture layers (each is one or more packages under `pkg/`):

- **Config (`pkg/config/`)** — parses the YAML config dir (`clusters`, `node-groups`, `bee-configs`, `checks`, `simulations`), resolves `_inherit` inheritance, and exports into the `orchestration.*Options` types. Read from a local dir or a Git repo (`config-git-repo`).
- **Orchestration (`pkg/orchestration/`)** — backend-agnostic `Cluster` → `NodeGroup` → `Node` model. The `k8s/` backend translates it into Kubernetes resources; `notset/` is the no-op fallback when K8s is disabled.
- **Checks (`pkg/check/`)** — each check implements the `beekeeper.Action` interface and is registered by name in the `Checks` map in `pkg/config/check.go`; `pkg/check/runner.go` resolves names to implementations and runs them.
- **Clients** — `pkg/bee/` (+ `pkg/bee/api/`) is the HTTP client for a running Bee node; `pkg/k8s/` wraps the Kubernetes client (tested with client-go's fake clientset; `pkg/k8s/mocks/` holds only the `ClientConfig`/RoundTripper doubles); `pkg/swap/` is the Geth/blockchain client.
- **Operational packages** — `stamper`, `nuker`, `restart`, `funder` act on already-running nodes and discover them through `pkg/node/` (`NodeProvider`: Beekeeper cluster, namespace+label, or Helm), not the orchestration layer.

## Deployment / operating modes

Commands work against clusters provisioned in three different ways — know which one applies:

1. **Beekeeper-managed Kubernetes** — Beekeeper creates the cluster from the config dir (`create bee-cluster`); checks/ops resolve nodes from the cluster definition.
2. **Static endpoints (no Kubernetes)** — set `enable-k8s: false` and point at static Bee node URLs in config (see `config/public-testnet-static.yaml`). Used for public-testnet checks.
3. **Externally-deployed clusters** — `stamper`/`nuke`/`restart`/`node-funder` target nodes by namespace/label or a Helm deployment (`--deployment-type=helm`), without a Beekeeper cluster definition.

## Conventions

- **Code style**: prefer clear, idiomatic Go that follows standard best practices and principles — small focused functions, explicit error wrapping with context, no premature abstraction. Keep changes minimal and consistent with the surrounding code.
- **Commits & PR titles**: Conventional Commits, lowercase type, no trailing period, `feat(scope): …` style (enforced by `commitlint.config.js`). Do not push commits; when a commit message is requested, use the subject line only — no body/description.
- **Tests**: prefer external test packages (`package foo_test`). Test the `pkg/k8s` clients against client-go's fake clientset instead of a live cluster — `fake.NewClientset()` with `PrependReactor` for error paths, and `watch.NewRaceFreeFake` + `PrependWatchReactor` for watch paths (buffer/drive events, bound tests with `context.WithTimeout`, never `time.Sleep`). The only hand-written mock left is `pkg/k8s/mocks` (the `ClientConfig`/`RoundTripper` doubles backing `pkg/k8s/k8s_test.go`). The race detector must pass.
- **Dependencies**: do not add or bump modules unless the task requires it (Dependabot handles routine bumps).
- **Linting**: `gofmt` + `gofumpt` formatting and the linters in `.golangci.yml` (errorlint, errname, nilerr, goconst, misspell, unconvert, copyloopvar) must pass. Note this repo does **not** use BSD copyright file headers — don't add them.

## Pre-commit checklist

Run before committing: `make build`, `make vet`, `make lint`, `make test` (use `make test-race` when touching concurrent code). Keep changes minimal and focused on the task.
