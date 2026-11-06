# Proposal: adopt-ateam-v038

Track: design (adds spec Requirements and touches both `agent-setup` and `merge-gate`; `design.md`).

Approved-by:

## Why

SemVocab still runs the pre-`ateam` Oct 16 harness carried from SemEngine: local `scripts/merge-check.sh` and
`scripts/openspec-queue.sh`, no `.mcp.json` or lifecycle hooks, no `.agents/review-policy.json` or host manifests, and
legacy `semvocab-{pickup,handoff,preflight}` skills. Meanwhile SemEngine (`C360Studio/semengine#446`, `#523`) and
SemTeams (`C360Studio/semteams#391`) adopted `ateam`, and `ateam v0.3.8` (`C360Studio/ateam#295`) adds `review_launch`,
`ateam review request`/`notify`, `.agents/hosts/{enjoy,herdr}.toml`, and the review-record pickup summary. Adopting
`v0.3.8` directly brings SemVocab to the full harness end state in one PR without an immediate bump.

## What Changes

- Run `ateam init` at `v0.3.8` to add `.mcp.json`, `.agents/hooks.json`, `.agents/review-policy.json`
  (`"review_launch": "agent"`), `.agents/hosts/{enjoy,herdr}.toml`, `.agents/plugins/ateam/`, base contracts
  `.agents/contracts/{architect,developer,reviewer}.md` (with the four `semvocab-*` contracts reading their base
  contract first), canonical skills (`pickup`, `handoff`, `preflight`, `design`, `review-pr`, `ponytail`), client
  settings under `.claude/`, `.codex/`, and `.gemini/`, `.gitignore` entries, and `openspec/config.yaml` design rules.
- Retire `.agents/skills/semvocab-{pickup,handoff}/`, `.claude/skills/semvocab-{pickup,handoff}/`, and
  `scripts/openspec-queue.sh`; drop the `python3` check from `scripts/doctor.sh`.
- Pin `ateam v0.3.8` in `scripts/doctor.sh`, `.github/workflows/ci.yml`, and `.github/workflows/merge-check.yml`.
- Add `ateam:doctor` (`ateam doctor --check`), `test:spec` (`ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"`),
  and `mutate:check` (`ateam mutate`) in `Taskfile.yml`; switch `spec:check` and `spec:queue` to `ateam state`; run
  `ateam:doctor` and `test:spec` in `scripts/verify.sh`.
- Split `Merge check` into `.github/workflows/merge-check.yml` with base-commit checkout, the `Merge check probe` job
  (`ateam review gate --awaiting`), and `scripts/merge-check.sh` calling `ateam review gate`
  (`ATEAM_REQUIRED_CHECKS=Required,Merge check`). Update `main` ruleset `25161512` at merge time to require both
  `Required` and `Merge check`.
- Apply the `ateam` protocol rules (`quick`/`design` track, 60-line `proposal.md` + `Approved-by:` + `OPEN-Q` hold,
  `review_launch: "agent"`, `ateam mutate`, `ateam test-spec --base`, Close hygiene, Ponytail footer + `// shortcut:`)
  across `.agents/protocol.md`, `AGENTS.md` (< 24,000 bytes), `.agents/rules-index.md`, and `.agents/README.md`.
- Replace retired script tests in `internal/harness/contract` with `ateam` parity, template-drift, workflow/wrapper, and
  task-wiring checks carrying paired sensitivity tests and `// spec:` citations.

## Capabilities

- Modified `agent-setup`: base+specialist contract layering, canonical skill + Gemini adapters, `ateam:doctor` +
  template parity, and brownfield `test:spec` citations.
- Modified `merge-gate`: `verify` gate order (`ateam:doctor`, `test:spec`), `ateam review gate` wrapper, separate
  `merge-check.yml` workflow with awaiting probe, two required checks (`Required`, `Merge check`), and `ateam state`
  spec queue/check wiring.

## Impact

- Repository harness, CI workflows, agent guidance/contracts/skills, OpenSpec specs, and `internal/harness/contract`
  tests only; zero changes to `catalog/`, `packs/`, `shacl/`, or `go.mod` runtime dependencies. Closes #36.
