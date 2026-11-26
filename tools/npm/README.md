# ports

Every dev server on your machine, by name — `http://shop.localhost` instead of `http://localhost:3000`, whether ports started it or not. See what's running, reach it by name, stop it safely. For people and coding agents.

```bash
npm i -g @pyjeebz/ports
ports proxy start          # names for every dev server, from login on
ports                      # what's running, by project and name
```

Or without installing: `npx @pyjeebz/ports here`.

This package runs the native `ports` binary for your platform, installed as an optional dependency (`@pyjeebz/ports-<os>-<arch>`). No postinstall script, so it works with `--ignore-scripts`.

Docs: https://ports.tools · Source: https://github.com/pyjeebz/ports
