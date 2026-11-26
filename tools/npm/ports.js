#!/usr/bin/env node
// The `ports` command npm installs: it runs the native binary from the
// platform package npm picked for this machine. Everything passes through
// untouched — arguments, stdio, the exit code — and a stop sent to this
// launcher reaches the binary, so killing `npx @pyjeebz/ports run …`
// really stops it.
"use strict";

const { spawn } = require("node:child_process");

const PLATFORMS = {
  "darwin arm64": "@pyjeebz/ports-darwin-arm64",
  "darwin x64": "@pyjeebz/ports-darwin-x64",
  "linux arm64": "@pyjeebz/ports-linux-arm64",
  "linux x64": "@pyjeebz/ports-linux-x64",
  "win32 x64": "@pyjeebz/ports-win32-x64",
};

const pkg = PLATFORMS[`${process.platform} ${process.arch}`];
if (!pkg) {
  console.error(
    `ports: no npm build for ${process.platform}/${process.arch} — see https://ports.tools/docs/installation`,
  );
  process.exit(1);
}

let bin;
try {
  bin = require.resolve(`${pkg}/bin/ports${process.platform === "win32" ? ".exe" : ""}`);
} catch {
  console.error(
    `ports: ${pkg} isn't installed. npm installs it as an optional dependency — ` +
      "reinstall without --omit=optional / --no-optional, or use: curl -fsSL https://ports.tools/install | sh",
  );
  process.exit(1);
}

const child = spawn(bin, process.argv.slice(2), { stdio: "inherit" });

// Ctrl-C reaches the binary from the terminal on its own; SIGINT is passed
// on only when there's no terminal — an agent or a script interrupting.
const forward = (signal) => {
  if (signal === "SIGINT" && process.stdin.isTTY) return;
  child.kill(signal);
};
const signals = ["SIGINT", "SIGTERM", "SIGHUP"];
for (const s of signals) process.on(s, forward);

child.on("error", (err) => {
  console.error(`ports: couldn't start ${bin}: ${err.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  for (const s of signals) process.removeListener(s, forward);
  if (signal) {
    process.kill(process.pid, signal); // end the way the binary ended
    return;
  }
  process.exit(code ?? 1);
});
