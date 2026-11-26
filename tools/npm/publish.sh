#!/bin/sh
# publish.sh <out-dir> [npm publish flags…]
#
# Publishes what build.mjs wrote: the platform packages first, then
# @pyjeebz/ports, so the main package never points at a version that isn't
# there yet. A version already on npm is skipped, so a rerun after a
# partial publish finishes the job instead of failing on the first one.
set -eu
out=$1
shift
for dir in "$out"/ports-*/ "$out"/ports/; do
  # resolved from here, so a relative and an absolute out-dir both work
  name=$(node -p "require(require('path').resolve(process.argv[1], 'package.json')).name" "$dir")
  version=$(node -p "require(require('path').resolve(process.argv[1], 'package.json')).version" "$dir")
  if npm view "$name@$version" version >/dev/null 2>&1; then
    echo "$name@$version is already on npm"
    continue
  fi
  (cd "$dir" && npm publish --access public "$@")
done
