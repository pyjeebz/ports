# ports

ports finds every dev server on your machine and gives it a name — `http://shop.localhost` instead of `http://localhost:3000`. Start servers the way you already do, from a terminal, an editor, `docker compose` or a coding agent: nothing to wrap, nothing to declare. See what's running, reach it by name, stop it safely. For people and coding agents.

```bash
npm i -g @pyjeebz/ports
ports                      # every dev server running, however it was started
ports proxy start          # each one at http://<name>.localhost, from login on
```

Or without installing: `npx @pyjeebz/ports here`.

This package runs the native `ports` binary for your platform, installed as an optional dependency (`@pyjeebz/ports-<os>-<arch>`). No postinstall script, so it works with `--ignore-scripts`.

Docs: https://ports.tools · Source: https://github.com/pyjeebz/ports
