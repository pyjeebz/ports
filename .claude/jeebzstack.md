# Ship config

How this repo goes from branch to users. Read by the `jeebzstack` plugin at every stage. Keep it true; it ships through PRs like everything else.

## Checks

The gate. Mirrors `.github/workflows/ci.yml` exactly, so a green local run means a green CI run. (CI also runs `test` on macOS and Windows; locally you only cover linux, so platform code in `internal/inspect` — `wsl.go`, `owner.go`, `service.go`, socket and home-path handling — leans on CI for the other two.)

- `go test ./internal/... ./cmd/...`
- `go build -v .`
- `go test ./test/...` (integration: builds the real binary, runs it against real listeners)
- `cmp install.sh web/app/install/script.sh && sh -n install.sh`
- `test -z "$(gofmt -l main.go cmd internal test tools)"`
- `go vet ./internal/... ./cmd/... ./test/... ./tools/...`
- `golangci-lint run ./internal/... ./cmd/... ./test/... ./tools/...` (CI pins v2.13.2)

Touching `web/`: `cd web && npm run lint && npm run build`. CI doesn't check the site at all.

## Prove it

- Build/run: `go build -o /tmp/ports . && /tmp/ports`
- Smoke: start a listener (`python3 -m http.server 8765` in some project dir), then `ports` should list `8765` with that project in the PROJECT column; `ports --json | jq '.[] | select(.port==8765)'` shows the same.
- Stops: `ports free 8765 --expect <project dir name>` frees it and reports what it stopped; `ports | grep 8765` should then be empty. Also check the refusal: `--expect python` must stop nothing and exit 1, because the process is `python3.12` and `--expect` matches exactly.
- Exit grammar: a bad flag or filter exits `2` (`ports --sort nope; echo $?`).
- Site: `cd web && npm run dev`, then load `/`, `/docs` and `/install`.

## Risky paths

Touching these makes a change **risky tier**: rollback written first, fresh-context review.

- `install.sh`, `web/app/install/script.sh`: strangers pipe it to `sh`. The two copies must be byte-identical (CI checks), and it's served live from ports.tools, so a broken one breaks every curl install until the next site deploy.
- `internal/inspect/kill.go`, `cmd/kill.go`, `cmd/free.go`: the only write path in a tool whose promise is "observer, never a runner". Stopping the wrong process is the worst bug ports can have. `--expect` exists to prevent exactly that.
- `internal/inspect/output.go` and any `--json`/`--format` change: the JSON shape is a contract for scripts. Additive only, unless it's a breaking release.
- `cmd/root.go` exit codes: the exit grammar (`2` = bad invocation) is documented and scripted against.
- `internal/inspect/config.go`: the `ports-config.yaml` format lives on users' disks. Old files must keep loading, or fail loudly naming the file.
- `.goreleaser.yaml`, `.github/workflows/release.yml`: the release pipeline, Homebrew cask and Scoop manifest. A mistake ships to package managers.
- Anything that starts, supervises or installs something: it crosses the settled "observer, never a runner" line in the README. Don't, without revisiting that decision explicitly.

## Deploy

Two independent things reach users:

**The binary.** Nothing deploys on merge; it reaches users by release (see below).

**ports.tools** (Next.js in `web/`). Vercel project `ports` (team `pyjeebzs-projects`) is **not** git-connected, so merging to main does not update the site.

- Trigger: manual, from `web/` on main after the merge: `vercel --prod`
- Check: `curl -sI https://ports.tools` returns 200, and `curl -fsSL https://ports.tools/install | cmp - install.sh` is silent (the live installer matches main)
- Rollback: `vercel rollback` (or promote the previous production deployment in the dashboard)
- Any merge touching `web/` **or `install.sh`** needs a site deploy, because the curl installer is served from the site.

## Release

- Trigger: push tag `vX.Y.Z` on a green main, which runs `release.yml` → goreleaser. Man pages are generated from the command tree at release time.
- The GitHub release is created as a **draft** (`release.draft: true`). Publish it after checking the assets: `gh release edit vX.Y.Z --draft=false`.
- Channels to verify: release assets (linux/darwin amd64+arm64, windows amd64, `checksums.txt`, man pages in each archive); `brew install pyjeebz/tap/ports` (cask in `pyjeebz/homebrew-tap`); `scoop install ports` (`pyjeebz/scoop-bucket`); `curl -fsSL https://ports.tools/install | sh`; `go install github.com/pyjeebz/ports@vX.Y.Z`. Each should report the new version from `ports --version`.
- Notes: GitHub release notes only (goreleaser `changelog: git`), rewritten from the merged PRs.
