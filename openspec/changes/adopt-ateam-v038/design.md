# Design: adopt-ateam-v038

Base: `b9ca45c`. Track: `design` (adds spec Requirements; modifies `agent-setup` and `merge-gate`). Issue: `#36`.

## Context

- `AGENTS.md:1-178` (14,884 bytes) and `.agents/rules-index.md` carry five rule domains without `ateam` gates.
- `.agents/README.md:95-115` and `.agents/contracts/semvocab-*.md` define four specialist contracts and three
  `semvocab-{pickup,handoff,preflight}` skills with no base contracts, `.mcp.json`, hooks, or `.gemini/` adapters.
- `Taskfile.yml:78-98`, `scripts/verify.sh:10`, and `scripts/doctor.sh:40-43` run `scripts/merge-check.sh` (271 lines)
  and `scripts/openspec-queue.sh` (298 lines, requiring `python3`), with no `ateam:doctor`, `test:spec`, or
  `mutate:check` targets.
- `.github/workflows/ci.yml:64-100` runs `merge-check` inside `ci.yml` from the PR merge ref and aggregates `verify` and
  `merge-check` into `Required`; ruleset `25161512` (`main-required-checks`) requires only `Required`.
- `internal/harness/contract/{agentsetup,mergecheck,specqueue}_test.go` test the pre-`ateam` layout and bash scripts.
- **Ownership & adopter seam inventory:** Harness-only change; zero `catalog/<domain>`, `packs/`, or `shacl/` symbols
  and zero SemEngine `vocabulary` overlap. No external Go consumer calls `scripts/` or `internal/harness/contract`;
  human and agent contributors (`claude`, `codex`, `gemini`) reach `Taskfile.yml` targets and GitHub Actions checks.
- **Overlapping open PRs & merge order:** Draft PR `#28` (`gemini/sensorml-pack`) edits `AGENTS.md`; whichever merges
  second merges `origin/main` and keeps `AGENTS.md` < 24,000 bytes and rule-index row parity. Dependabot PRs `#29` and
  `#34` touch `package.json`/`package-lock.json` (`scripts/doctor.sh:13` reads versions dynamically) and can merge
  either before `#36` or after under `ateam review gate`'s bot-author rule.

## Goals / Non-Goals

- **Goals:** Adopt `ateam v0.3.8` in one PR (bootstrap configs, hooks, MCP, host manifests, `review_launch: "agent"`,
  base+specialist contracts, canonical skills, `ateam review gate` + `Merge check probe`, `ateam state` spec queue/check,
  `ateam mutate`, `ateam test-spec --base`, updated protocol/rules/tests).
- **Non-Goals:** Backfilling `// spec:` comments onto untouched pre-adoption tests; touching ontology catalogs, packs,
  or SHACL code; adding network/Docker harness scripts SemVocab does not use.

## Decisions

- **D1. Single-PR `v0.3.8` adoption vs four-PR staged rollout.**
  - *Option A (chosen):* Land bootstrap, merge-gate, spec-queue, and protocol rules together at `v0.3.8`. Cost: larger
    diff (~45 files). Blast radius: harness/CI/docs only. Forecloses bisecting intermediate `v0.3.3`–`v0.3.7` states.
  - *Option B:* Mirror SemEngine's four PRs (`#446`). Cost: four cross-agent review cycles on a small repo. Forecloses
    immediate `v0.3.8` host-manifest availability until PR D.
  - *Adopter seam:* Contributors need `ateam v0.3.8` on `PATH` (`GOTOOLCHAIN=auto go install
    github.com/c360studio/ateam/cmd/ateam@v0.3.8`); without it `task doctor` and `task verify` fail naming the command.
- **D2. Specialist contracts layered on `ateam` base contracts vs replacing `semvocab-*` contracts.**
  - *Option A (chosen):* Keep `semvocab-{architect,developer,reviewer,technical-writer}.md`; the first three instruct
    reading `.agents/contracts/{architect,developer,reviewer}.md` first, and `.agents/contracts/reviewer.md` is the sole
    home of the `Review record` grammar. Cost: maintaining four domain files. Forecloses single-file contract lookup.
  - *Option B:* Drop `semvocab-*` contracts for generic `ateam` contracts. Cost: loses SemEngine IRI/registry invariants
    (`AGENTS.md` § 1–4). Forecloses automated parity with `.claude/agents/` and `.codex/agents/`.
  - *Adopter seam:* Existing `semvocab-*` subagent names keep working unchanged; doing nothing breaks no invocation.
- **D3. Separate `merge-check.yml` with `Merge check probe` + `review_launch: "agent"` vs keeping `merge-check` in
  `ci.yml`.**
  - *Option A (chosen):* `.github/workflows/merge-check.yml` checks out `github.event.pull_request.base.sha`, runs
    `ateam review gate --awaiting` in `awaiting`, skips/renames `merge-check` on drafts or `awaiting=true`, and runs
    `scripts/merge-check.sh` (`ATEAM_REQUIRED_CHECKS=Required,Merge check`). `.agents/review-policy.json` sets
    `"review_launch": "agent"` so sessions can invoke `ateam review request`/`notify` with `.agents/hosts/enjoy.toml`.
    Cost: one ruleset `25161512` edit at merge time. Forecloses single-check rulesets.
  - *Option B:* Keep `merge-check` inside `ci.yml`. Cost: PR description edits rerun `Verify`, and PR branches could
    self-modify `merge-check.sh`. Forecloses base-commit tamper resistance and awaiting-review skip behavior.
  - *Adopter seam:* A ready code PR with no review shows `Merge check` absent (blocking merge without false red); once a
    `Review record` comment lands, rerunning `Merge check` evaluates it. Doing nothing leaves the PR unmergeable.
- **D4. Brownfield `ateam test-spec --base` + `ateam mutate` vs deferring `test-spec`.**
  - *Option A (chosen):* `task test:spec` runs `ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"` inside
    `task verify`, and `task mutate:check` wraps `ateam mutate`. Cost: new/edited tests must add `// spec:` comments.
    Forecloses adding uncited tests without a spec Requirement.
  - *Option B:* Defer `test-spec` in `protocol.md` as SemEngine `#447` did before `ateam#237`. Cost: spec drift on new
    tests. Forecloses automated spec-citation checks in `verify`.
  - *Adopter seam:* Touching any `Test*`/`Fuzz*`/`Example*` without `// spec: openspec/specs/<cap>/spec.md#Requirement:
    <name>` fails `task test:spec` naming the test; untouched tests need no edit.

## Risks / Trade-offs

- **Ruleset cutover:** Adding `Merge check` to ruleset `25161512` before `#36` merges would block PRs lacking the
  workflow; updating `25161512` immediately at `#36` land time (with `ATEAM_REQUIRED_CHECKS` verified once updated)
  avoids a deadlocked window.
- **Design length warning:** Ontology port designs exceed 120 lines due to mandatory provenance inventories; accepting
  `ateam state spec-check`'s non-fatal warning preserves provenance depth.

## Migration Plan

1. Approve this proposal/design (`Approved-by:` filled and task `1.1` `OPEN-Q` hold removed) and wait for the `ateam
   v0.3.8` tag.
2. Install `ateam v0.3.8`, run `ateam init`, layer contracts/skills/configs, update scripts/workflows/protocol/tests,
   pass `task verify` cleanly, archive `adopt-ateam-v038`, obtain cross-agent review, update ruleset `25161512`, and
   merge.

## Open Questions

- **Owner ruling (`OPEN-Q`):** Approve `proposal.md` and `design.md` (`Approved-by:` in `proposal.md`) so implementation
  proceeds as soon as `C360Studio/ateam` tags `v0.3.8`.
