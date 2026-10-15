# phase-0-agent-setup

## Why

SemVocab has only a license (`git log`: `bc382ba Initial commit`). Issue #1 Phase 0 asks for `go.mod`,
`Taskfile.yml`, a repository `AGENTS.md`, pinned tools, and reproducible verification commands before any catalog is
ported. SemVocab is worked on by Claude, Codex, and Gemini sessions, as SemEngine is, and the owner chose (2026-10-15,
on this change's claim PR) to carry SemEngine's whole process: role contracts, cross-agent review, `merge-check`, a
required CI check, the known-flake gate, and a ruleset on `main`. This change carries that setup and adapts it to a
vocabulary-extension library; it ports no catalog.

## What Changes

- Agent guidance: `AGENTS.md` (SemVocab purpose, commands, five rule domains), `CLAUDE.md` importing it,
  `.agents/protocol.md`, `.agents/rules-index.md`, `.agents/README.md`.
- Four role contracts under `.agents/contracts/` (architect, developer, reviewer, technical writer), with thin Claude
  (`.claude/agents/`) and Codex (`.codex/agents/`) adapters on SemEngine's model and effort routing.
- Skills `semvocab-preflight`, `semvocab-handoff`, `semvocab-pickup` with Claude adapters. SemEngine's NATS and graph
  capability skills are not carried.
- Tooling: `go.mod` (Go 1.26.9; revive and govulncheck as tool directives), `.task-version`, `.nvmrc`,
  `package.json` (OpenSpec, markdownlint), `revive.toml`, markdownlint configs, `Taskfile.yml`, and scripts `doctor`,
  `verify`, `gopkgs`, `openspec-queue`, `merge-check`.
- CI: `verify`, `merge-check`, `required`; dependabot for actions, npm, and gomod.
- Contract tests under `internal/harness/contract`: `TestAgentSetupParity` and its sensitivity test, and the carried
  tests of `merge-check.sh` and `openspec-queue.sh`.
- GitHub: labels `type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`; a `main` ruleset requiring the
  `Required` check on an up-to-date branch.

## Not carried, and why

- SemEngine's integration lane, NATS fixture, Docker naming guards, `lint-test-ports.sh`, and
  `cleanup-roots-check.sh`: SemVocab has no networked or Docker-backed code.
- `cover:check`: no package has a coverage floor yet; the catalog work adds targets.
- `ledger:check`, `pindiff`, and `mutcheck`: deferred to the catalog work, which writes `docs/admission-ledger.yaml`.
- SemEngine's capability skills (`entity-or-bucket`, `kv-or-stream`, `new-payload`, `query-pattern`).

## Impact

No product code. Later Phase 0 work on #1 (admission ledger, catalog dispositions, the Phase 0 OpenSpec change with
capability specs) builds on this setup.
