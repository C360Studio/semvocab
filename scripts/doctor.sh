#!/usr/bin/env bash
# Report tool versions and compare each against its single pin. Starts no workloads.
set -uo pipefail
cd "$(dirname "$0")/.."

fail=0
ok()   { printf '  ok    %-13s %s\n' "$1" "$2"; }
bad()  { printf '  FAIL  %-13s %s\n' "$1" "$2"; fail=1; }
warn() { printf '  warn  %-13s %s\n' "$1" "$2"; }

# version_ge A B: true when A >= B (dotted numeric versions).
version_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]; }
pkg_pin() { node -p "require('./package.json').devDependencies['$1']" 2>/dev/null; }

echo "SemVocab doctor"

# go: go.mod's go directive is the pin; the toolchain must be at least that.
if command -v go >/dev/null; then
  want=$(awk '$1=="go"{print $2; exit}' go.mod)
  have=$(go env GOVERSION | sed 's/^go//')
  if version_ge "$have" "$want"; then ok go "$have (go.mod: $want)"; else bad go "$have < go.mod $want"; fi
else bad go "not found"; fi

# task: .task-version is the pin (CI installs it from the same file).
if command -v task >/dev/null; then
  want=$(tr -d '[:space:]' < .task-version)
  have=$(task --version | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' | head -n1)
  if [ "$have" = "$want" ]; then ok task "$have"; else bad task "$have != .task-version $want"; fi
else bad task "not found"; fi

# node: .nvmrc is the pin, compared by major version.
if command -v node >/dev/null; then
  want=$(tr -d '[:space:]v' < .nvmrc)
  have=$(node --version | sed 's/^v//')
  if [ "${have%%.*}" = "${want%%.*}" ]; then ok node "$have (.nvmrc: $want)"; else bad node "$have != .nvmrc $want"; fi
else bad node "not found"; fi

if command -v npm >/dev/null; then ok npm "$(npm --version)"; else bad npm "not found"; fi

# ateam: every go install line in ci.yml and merge-check.yml names the one pinned version, and each
# file has one; task ateam:doctor runs it.
semver='v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?'
workflows=(.github/workflows/ci.yml .github/workflows/merge-check.yml)
missing=""
for f in "${workflows[@]}"; do
  grep -Eq "ateam/cmd/ateam@$semver" "$f" 2>/dev/null || missing="$missing $f"
done
pins=$(cat "${workflows[@]}" 2>/dev/null | grep -Eo "ateam/cmd/ateam@$semver" | sed 's/.*@//' | sort -u || true)
npins=$(printf '%s' "$pins" | grep -c . || true)
if [ -n "$missing" ]; then
  bad ateam "no ateam install line in$missing"
elif [ "$npins" -ne 1 ]; then
  bad ateam "ci.yml and merge-check.yml ateam install lines name $npins versions ($(echo $pins)), want one"
elif command -v ateam >/dev/null; then
  have=$(ateam version 2>&1 | grep -Eo "$semver" | head -n1 || true)
  if [ "$have" = "$pins" ]; then ok ateam "$have"; else bad ateam "${have:-unknown} != workflow pin $pins (put \$(go env GOPATH)/bin first on PATH)"; fi
else bad ateam "not found (GOTOOLCHAIN=auto go install github.com/c360studio/ateam/cmd/ateam@$pins)"; fi

# npm-installed tools resolve from node_modules, pinned in package.json.
check_npm_tool() { # name bin
  local want have
  want=$(pkg_pin "$1")
  if [ ! -x "node_modules/.bin/$2" ]; then bad "$2" "not installed: run npm ci"; return; fi
  have=$(node -p "require('./node_modules/$1/package.json').version")
  if [ "$have" = "$want" ]; then ok "$2" "$have"; else bad "$2" "$have != package.json $want"; fi
}
check_npm_tool @fission-ai/openspec openspec
check_npm_tool markdownlint-cli2 markdownlint-cli2

if [ "$fail" -ne 0 ]; then echo "doctor: FAILED"; exit 1; fi
echo "doctor: ok"
