# ports

**See what's running on your machine, by project — and kill it by name.**

`ports` answers one question, instantly and with zero configuration: *what is running on this machine right now, and what does it belong to?*

```
$ ports
PORT   PROCESS      PROJECT              UPTIME
3000   Next.js      ~/projects/shop      14m
5432   PostgreSQL   docker: shop-db      2d
8000   FastAPI      ~/projects/shop      14m
5173   Vite         ~/projects/admin     3h

$ ports kill shop
✓ stopped Next.js (:3000)
✓ stopped FastAPI (:8000)

$ ports free 3000
✓ killed Next.js (pid 48391) — port 3000 is free
```

## Why

Every developer knows this dance:

```
Error: Port 3000 is already in use.

$ lsof -i :3000
COMMAND   PID
node      48391

$ kill -9 48391
```

And still nobody knows what PID 48391 *was*. React? Next.js? A dev server from
yesterday? The database you actually needed? Ports and PIDs are how the
operating system sees your machine — they are not how *you* see it. You see
projects: a frontend, an API, a database that belong together.

Existing tools (`lsof`, `kill-port`, `fkill`) stop at the PID layer — they can
kill a number, but they can't tell you what the number is. `ports` exists to
add the missing context: which framework a process is, which project directory
it belongs to, which container is really behind `docker-proxy`, how long it has
been running. That context is what turns "kill a random PID and hope" into
"stop yesterday's dev server, keep the database."

## What it does

- **Lists** every listening port with the framework, project directory, and uptime — not just a PID
- **Groups** processes into the applications they belong to, so `frontend :3000`, `api :8000`, and `db :5432` read as one project
- **Unwraps Docker**, naming the real container and compose service instead of `docker-proxy`
- **Stops things safely by name** — `ports kill <project>`, `ports free <port>` — with graceful termination before force
- **Finds things** — `ports find postgres` → `5432`
- **Speaks JSON** (`--json`) so scripts and coding agents can ask what's running

## Principles

- **Observer, never a runner** — it never starts, supervises, or installs anything. Read-only plus targeted kill. There are plenty of process managers; this is not one, and it never will be.
- **Zero config** — no manifest, no daemon, nothing to declare. All context is inferred from the machine itself. Useful sixty seconds after install, on a machine in whatever state it's in.
- **Local-first** — a single static binary. No cloud, no accounts, no telemetry.

## Install

**Homebrew** (macOS):

```
brew install pyjeebz/tap/ports
```

**Scoop** (Windows):

```
scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket
scoop install ports
```

**curl** (Linux, macOS):

```
curl -fsSL https://raw.githubusercontent.com/pyjeebz/ports/main/install.sh | sh
```

Or grab a binary from the [releases page](https://github.com/pyjeebz/ports/releases), or `go install github.com/pyjeebz/ports@latest`.
