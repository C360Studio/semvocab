#!/usr/bin/env bash
# Regenerate catalog/bfo/testdata/semstreams-bfo.txt, the SemStreams-pin parity
# fixture TestCatalogPortParity reads, from a local SemStreams clone. Reads the
# frozen pin with `git show`; checks nothing out and imports no SemStreams code.
# Usage: scripts/gen-semstreams-fixture.sh <path to a github.com/C360Studio/semstreams clone>
set -euo pipefail
pin=8b99efe9c66a4faa4fa509f9f62cc6bad8392128
src=${1:?usage: $0 <path to a semstreams clone>}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
git -C "$src" show "$pin:vocabulary/bfo/bfo.go" >"$tmp/bfo.go"
git -C "$src" show "$pin:vocabulary/bfo/hierarchy.go" >"$tmp/hierarchy.go"
cd "$(dirname "$0")/.."
go run ./internal/tools/semstreamsfixture -pin "$pin" "$tmp/bfo.go" "$tmp/hierarchy.go" \
  >catalog/bfo/testdata/semstreams-bfo.txt
echo "wrote catalog/bfo/testdata/semstreams-bfo.txt"
