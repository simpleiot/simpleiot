#!/usr/bin/env bash
#
# Build the Elm SPA frontend and emit a gzipped elm.js.
#
#   Usage: build_frontend.sh <frontend_src_dir> <output_elm_js_gz>
#
# This is a deliberately non-hermetic action: it shells out to the host's
# node/npm and the Elm toolchain (installed on demand via elm-tooling), relying
# on the ~/.npm and ~/.elm caches. HOME and PATH are passed through to the
# action via .bazelrc so those caches and binaries resolve. The Elm sources are
# declared inputs, so editing them invalidates this action and rebuilds anything
# that embeds the result (the frontend go_library, and therefore the binary).
set -euo pipefail

src_dir=$1
out=$2

# Resolve to absolute paths before changing directories.
src_dir=$(cd "$src_dir" && pwd)
out_dir=$(cd "$(dirname "$out")" && pwd)
out="$out_dir/$(basename "$out")"

: "${HOME:?HOME is not set; ensure 'build --action_env=HOME' is in .bazelrc}"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# Copy the inputs out of the Bazel symlink tree into a clean working directory
# so the npm/elm-spa build can write node_modules and .elm-spa freely.
cp -RL \
	"$src_dir/src" \
	"$src_dir/elm.json" \
	"$src_dir/elm-tooling.json" \
	"$src_dir/package.json" \
	"$src_dir/package-lock.json" \
	"$work/"
mkdir -p "$work/public/dist"

cd "$work"
npm ci --no-audit --no-fund --silent
npx --no-install elm-tooling install
npx --no-install elm-spa build

# -n: omit timestamp/name from the gzip header so the output is reproducible.
gzip -nf public/dist/elm.js

cp public/dist/elm.js.gz "$out"
