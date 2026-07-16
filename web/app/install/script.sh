#!/bin/sh
# ports installer — https://ports.tools
# usage: curl -fsSL https://ports.tools/install | sh
set -eu

REPO="pyjeebz/ports"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$os" in
  linux | darwin) ;;
  *)
    echo "unsupported OS: $os (on Windows, use scoop: scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket && scoop install ports)" >&2
    exit 1
    ;;
esac
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" |
  grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)
[ -n "$tag" ] || { echo "could not resolve the latest release" >&2; exit 1; }
version=${tag#v}

name="ports_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "downloading ports $version ($os/$arch)..."
curl -fsSL -o "$tmp/$name" "$base/$name"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp" && grep " $name\$" checksums.txt | sha256sum -c -) >/dev/null
else
  (cd "$tmp" && grep " $name\$" checksums.txt | shasum -a 256 -c -) >/dev/null
fi

bindir="${PORTS_BIN:-}"
if [ -z "$bindir" ]; then
  if [ -w /usr/local/bin ]; then bindir=/usr/local/bin; else bindir="$HOME/.local/bin"; fi
fi
mkdir -p "$bindir"
tar -xzf "$tmp/$name" -C "$tmp" ports
install -m 0755 "$tmp/ports" "$bindir/ports"

echo "✓ installed ports $version to $bindir/ports"
case ":$PATH:" in
  *":$bindir:"*) ;;
  *) echo "note: $bindir is not on your PATH" ;;
esac
