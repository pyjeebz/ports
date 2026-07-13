# ports — project context

**One-liner:** See what's running on your machine, by project — and kill it by name.

`ports` is a zero-config CLI that lists every listening TCP port with the context a developer recognizes — framework (Next.js, FastAPI, PostgreSQL…), project directory, container behind docker-proxy, uptime — and stops things safely by name (`ports kill shop`, `ports free 3000`, `ports find postgres`).

Project home (milestones, roadmap, decision log): https://app.notion.com/p/39b72340cd57816b9da4dd70795e09b9

## Hard lines — never violate these

- **Observer, never a runner.** Never start, restart, supervise, or install anything. Read-only + targeted kill only. No `ports run`, ever.
- **Zero config.** No manifest, no config file, no daemon. All context is *inferred* (process cwd, cmdline, package.json, Docker API) — never declared by the user.
- **Local-first.** Single static binary. No cloud, accounts, or telemetry.

Every feature must be additive context or output under the same contract.

## Stack (locked)

- Go (single static binary, easy cross-compile). Module: `github.com/pyjeebz/ports`.
- `gopsutil/v4` is the only scan dependency (`net.Connections("tcp")` + `process` for name/cmdline/cwd/create-time).
- CLI: stdlib dispatch in `main.go` (no cobra). Output: `text/tabwriter` table + `--json`.
- Docker enrichment via raw HTTP over the unix socket (`/containers/json`, `POST /containers/{id}/stop`) — do NOT import the Docker SDK.

## Architecture

Pipeline: **collect** (gopsutil: listening sockets → PIDs) → **enrich** (framework detection from exe/cmdline heuristics; project from process cwd walked up to git root or nearest marker file; Docker port→container/compose-service mapping) → **render** (table / JSON) or **act** (kill/free/find).

- `internal/inspect` — `Service` model, `Scan()`, framework rules (table-driven, cheap to extend), project detection, docker socket client
- `main.go` — dispatch: default list, `find <q>`, `free <port>`, `kill <q>`, `--json`, `--force`

Details that matter:
- Dedupe listeners by (pid, port) — v4/v6 duplicates.
- Non-root can't see other users' PIDs (Pid=0): show `?`, hint about sudo; Docker enrichment fills root-owned docker-proxy rows via the socket, no sudo needed.
- Kill = SIGTERM → wait ~2s → SIGKILL (`--force` = immediate). Docker-backed ports: stop the *container* via the API, never kill docker-proxy.
- Project grouping: outermost `.git` wins (groups monorepos); else nearest marker (package.json, go.mod, pyproject.toml, …). Home dir itself is never a project.

## Milestones

- M1 — scan + context table (`ports`, `--json`)
- M2 — framework detection + project grouping
- M3 — actions: `kill` / `free` / `find`
- M4 — Docker unwrap (compose project/service labels)
- M5 — WSL2 awareness (Windows-side vs WSL listeners) + goreleaser release

Status: scaffold only (go.mod + README). Nothing implemented yet — start at M1.

## Conventions

- Commits: terse one-liners, just what was committed (e.g. `m1: scan + table`). No bodies, no trailers.
- Solo repo: commit/push straight to `main`, no branches/PRs — and only when asked.
- Keep the README free of milestones — those live in Notion.
