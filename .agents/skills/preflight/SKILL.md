---
name: preflight
description: >-
  Select and run the existing SemVocab verification gates for a concrete diff and record their evidence.
  Use before implementation pushes or when assessing local readiness.
---

# Verify a SemVocab change

The [shared protocol](../../protocol.md) owns claims and landing. Read it first. This skill selects existing checks and
records evidence; it grants no merge, tag or cleanup authority.

## 1. Establish the diff and its owner

Work in the claim's worktree. Verify its branch, HEAD, dirty/untracked files, upstream and PR target before choosing
checks. Compare the committed diff against the actual PR base, including a non-default target in a stack; also inspect
the staged and unstaged work. Do not assume the discovery checkout's local `main` is current. Before every commit or
push, verify the branch still matches the claimed PR head.

## 2. Select the existing gates

One Task entrypoint serves local work, MCP (`ateam_task`), and CI. Put `$(go env GOPATH)/bin` first on `PATH` so `ateam`
resolves to the pinned `v0.3.8` binary (`task doctor` checks it). `task --list` shows what exists; inspect
`Taskfile.yml` and the workflows under `.github/` when exact coverage matters.

| Task | What it does |
| --- | --- |
| `task fmt` | Formats Go sources; the only task here that changes files |
| `task fmt:check` | Fails if any Go source needs formatting; writes nothing |
| `task doctor` | Reports tool versions against their pins (`go`, `task`, `node`, `npm`, `ateam`, `openspec`, `markdownlint-cli2`); starts no workloads |
| `task ateam:doctor` | Runs `ateam doctor --check` with session environment variables cleared to verify `.agents/`, `.claude/`, `.codex/`, `.gemini/`, `.mcp.json`, and `CLAUDE.md` parity without mutating the tree |
| `task spec:check` | Validates all OpenSpec changes and specs strictly, then runs `ateam state spec-check` (fails on missing `config.yaml`/canonical sections or a `Hold:` outside tasks; warns when `proposal.md` > 60 or `design.md` > 120 lines) |
| `task spec:queue` | Runs `ateam state spec-queue` to show each in-flight OpenSpec change with its holds and blocking conditions |
| `task test:spec` | Runs `ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"`, requiring valid `// spec:` citations on new or modified Go tests since the merge base and refusing broken citations anywhere |
| `task docs:check` | Lints Markdown with `markdownlint-cli2` |
| `task tidy:check` | Fails if `go.mod` or `go.sum` are not tidy |
| `task build` | Builds all packages |
| `task vet` | `go vet` |
| `task lint` | Pinned `revive` |
| `task vuln` | Pinned `govulncheck`; review findings against reachable behavior |
| `task test:unit` | Unit tests once with the race detector at one CPU |
| `task test:repeat` | Unit tests five times at one CPU without the race detector, in shuffled order; `-- <pkgs>` to focus |
| `task mutate:check -- -pkg … -test … -file … -mutant … -expect …` | Runs `ateam mutate` by hand through `go test -overlay` (writes nothing in the repository) and fails unless the verdict is `detection` |
| `task merge:check -- <n>` | Runs `scripts/merge-check.sh` (`ateam review gate` at the pin): checks `main` ruleset up-to-date status checks, `implemented-by:` and `reviewed-by:` with `(model, effort)` and non-weaker cross-provider review record, and open `class:flake` issues; reads GitHub |
| `task verify` | Runs `ateam:doctor`, `spec:check`, `test:spec`, `docs:check`, `fmt:check`, `tidy:check`, `build`, `vet`, `lint`, `vuln`, `test:unit`, `test:repeat` in that order, printing timings and failing on tracked-file drift |

CI has two workflows: in `ci.yml`, `verify` runs `task doctor` and `task verify`, and `required` (`Required`) fails
unless `verify` succeeded; `merge-check.yml` runs `ateam review gate --awaiting` in `Merge check probe` and
`scripts/merge-check.sh` as `Merge check` from the base commit on non-draft pull requests.

Select by what the diff changes:

- **Documentation or skill instructions only:** `task fmt:check`, `task docs:check`, and `git diff --check`; add
  `task spec:check` when `openspec/` changed.
- **Agent setup (`AGENTS.md`, `CLAUDE.md`, `.agents/`, `.claude/`, `.codex/`, `.gemini/`, `.mcp.json`):**
  `task ateam:doctor`, `task docs:check`, and `task test:unit`.
- **An OpenSpec change:** `task spec:check`, then read `task spec:queue` in this worktree.
- **Go behavior or tests:** `task test:spec`, `task test:unit`, `task vet`, `task lint`, `task build`; run
  `task mutate:check` on non-trivial branches or refusals.
- **Pack YAML/JSON, Turtle, or catalog data:** the Go gates, since the tests that load them are the check.
- **Dependencies or `go.mod`:** `task tidy:check` and `task vuln` in addition to the Go gates.
- **Before any implementation push:** `task verify`.
- **Immediately before merging:** `task merge:check -- <n>`, in a shell where `GITHUB_ACTIONS` is not set.

Do not hand-run a narrower command in place of a gate CI runs: the task owns flags and pins.

## 3. Protect shared test infrastructure

Serialize heavy work on a shared host. Preserve other sessions' resources and worktrees. Cleanup is limited to an
identified completed or abandoned run this session is authorized to clean up.

## 4. Record Evidence and remaining gates

Record the tested HEAD (or dirty snapshot), exact command, exit status, assertions/tests actually exercised, and
log/artifact location, plus the 6-line testing block (`Decision`, `Rule`, `Inputs`, `Run`, `Mutation`, `Not covered`)
when touching parsers, validators, or laws, ending with Ponytail's 1–2 line `Skipped:` / `Residual risk:` summary.
When the implementation changes, older green results remain evidence for the older revision.

Resolve failures from evidence; a successful re-run is not a fix for a flake. A red that your diff does not explain is
filed as an issue before the next push, labelled `class:flake` when it passes and fails on the same tree. An open flake
is fixed; `Merge check` fails every pull request that does not close it.

After local verification, assess the PR's current-head hosted checks and the protocol's review/archive gates. Local
green is not merge authorization.
