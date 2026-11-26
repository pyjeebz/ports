#!/usr/bin/env node
// build.mjs <version> <archives-dir> <out-dir>
//
// Turns a release's archives into the npm packages: one per platform,
// holding that platform's binary and nothing else, and @pyjeebz/ports,
// which depends on all five optionally — npm installs only the one whose
// os and cpu match — and runs it through a small launcher. No postinstall
// script, so installs work with --ignore-scripts. Every archive is checked
// against the release's checksums.txt before its binary is taken.

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { chmodSync, copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const [version, archives, out] = process.argv.slice(2);
if (!version || !archives || !out || !/^\d+\.\d+\.\d+/.test(version)) {
  console.error("usage: build.mjs <version, e.g. 0.2.0> <archives-dir> <out-dir>");
  process.exit(2);
}

const SCOPE = "@pyjeebz";
const PLATFORMS = [
  { goos: "darwin", goarch: "arm64", os: "darwin", cpu: "arm64", ext: "tar.gz" },
  { goos: "darwin", goarch: "amd64", os: "darwin", cpu: "x64", ext: "tar.gz" },
  { goos: "linux", goarch: "arm64", os: "linux", cpu: "arm64", ext: "tar.gz" },
  { goos: "linux", goarch: "amd64", os: "linux", cpu: "x64", ext: "tar.gz" },
  { goos: "windows", goarch: "amd64", os: "win32", cpu: "x64", ext: "zip" },
];

const common = {
  version,
  license: "MIT",
  homepage: "https://ports.tools",
  repository: { type: "git", url: "git+https://github.com/pyjeebz/ports.git" },
  bugs: "https://github.com/pyjeebz/ports/issues",
};

const sums = Object.fromEntries(
  readFileSync(join(archives, "checksums.txt"), "utf8")
    .trim()
    .split("\n")
    .map((l) => l.trim().split(/\s+/).reverse()),
);

rmSync(out, { recursive: true, force: true });
const optionalDependencies = {};

for (const p of PLATFORMS) {
  const archive = `ports_${version}_${p.goos}_${p.goarch}.${p.ext}`;
  const data = readFileSync(join(archives, archive));
  const sum = createHash("sha256").update(data).digest("hex");
  if (sums[archive] !== sum) {
    console.error(`${archive}: checksum ${sum} doesn't match checksums.txt (${sums[archive] ?? "missing"})`);
    process.exit(1);
  }

  const name = `${SCOPE}/ports-${p.os}-${p.cpu}`;
  const dir = join(out, `ports-${p.os}-${p.cpu}`);
  const exe = p.os === "win32" ? "ports.exe" : "ports";
  mkdirSync(join(dir, "bin"), { recursive: true });

  const tmp = mkdtempSync(join(tmpdir(), "ports-npm-"));
  if (p.ext === "zip") execFileSync("unzip", ["-q", join(archives, archive), exe, "-d", tmp]);
  else execFileSync("tar", ["-xzf", join(archives, archive), "-C", tmp, exe]);
  copyFileSync(join(tmp, exe), join(dir, "bin", exe));
  chmodSync(join(dir, "bin", exe), 0o755);
  rmSync(tmp, { recursive: true, force: true });

  writeFileSync(
    join(dir, "package.json"),
    JSON.stringify(
      {
        name,
        ...common,
        description: `The ports binary for ${p.os} ${p.cpu}. Install @pyjeebz/ports instead.`,
        os: [p.os],
        cpu: [p.cpu],
        files: ["bin"],
        preferUnplugged: true,
      },
      null,
      2,
    ) + "\n",
  );
  writeFileSync(join(dir, "README.md"), `# ${name}\n\nThe \`ports\` binary for ${p.os} ${p.cpu}. Install [\`@pyjeebz/ports\`](https://www.npmjs.com/package/@pyjeebz/ports) instead — it picks this package for you.\n`);
  optionalDependencies[name] = version;
  console.log(`${name}@${version}  ${archive}`);
}

const main = join(out, "ports");
mkdirSync(join(main, "bin"), { recursive: true });
copyFileSync(join(here, "ports.js"), join(main, "bin", "ports.js"));
chmodSync(join(main, "bin", "ports.js"), 0o755);
copyFileSync(join(here, "README.md"), join(main, "README.md"));
writeFileSync(
  join(main, "package.json"),
  JSON.stringify(
    {
      name: `${SCOPE}/ports`,
      ...common,
      description: "Every dev server on your machine, by name — http://shop.localhost, not http://localhost:3000.",
      keywords: ["ports", "localhost", "dev-server", "proxy", "cli", "mcp", "agents"],
      bin: { ports: "bin/ports.js" },
      files: ["bin"],
      engines: { node: ">=18" },
      optionalDependencies,
    },
    null,
    2,
  ) + "\n",
);
console.log(`${SCOPE}/ports@${version}`);
