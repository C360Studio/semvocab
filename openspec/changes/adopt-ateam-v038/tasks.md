# Tasks: adopt-ateam-v038

Each task ends when `task verify` passes on a clean tree with `ateam v0.3.8` first on `PATH`.

## 1. Bootstrap, contracts, and skills

- [x] 1.1 Approve `proposal.md` and `design.md` (`Approved-by:` filled) for `#36` (`AGENTS.md:147-152`,
  `.agents/protocol.md:47-53`). Acceptance: `proposal.md` carries `Approved-by: <name> (<YYYY-MM-DD>)` and this hold is
  removed.
- [x] 1.2 Install `ateam v0.3.8` and run `ateam init` on the worktree, setting `"review_launch": "agent"` in
  `.agents/review-policy.json`, preserving `.agents/hosts/{enjoy,herdr}.toml`, adding `.ateam/` and
  `.claude/settings.local.json` to `.gitignore:1-26`, adding design rules to `openspec/config.yaml:1-33`, and ignoring
  unmodified `ateam` templates and `.worktrees/**`/`.ateam/**` in `.markdownlint-cli2.yaml:5-12`. Acceptance:
  `ateam doctor --check` passes on the seeded files and `task docs:check` reports zero errors.
- [x] 1.3 Layer `.agents/contracts/semvocab-{architect,developer,reviewer}.md` onto
  `.agents/contracts/{architect,developer,reviewer}.md` (moving review-record grammar ownership to `reviewer.md`), adapt
  `.agents/skills/preflight/SKILL.md` with SemVocab's gates while preserving `ateam`'s section markers, and delete
  `.agents/skills/semvocab-{pickup,handoff,preflight}/` and `.claude/skills/semvocab-{pickup,handoff}/`
  (`.agents/README.md:95-115`). Acceptance: `env -u ATEAM_SESSION_ID -u CLAUDE_SESSION_ID -u CODEX_SESSION_ID -u
  GEMINI_SESSION_ID ateam doctor --check` exits 0.

## 2. Taskfile, scripts, and CI workflows

- [x] 2.1 Pin `ateam v0.3.8` in `scripts/doctor.sh:1-58` (dropping the `python3` check at lines 40-43), delete
  `scripts/openspec-queue.sh:1-298`, update `Taskfile.yml:78-103` (`ateam:doctor`, `test:spec`, `mutate:check`,
  `spec:check`, `spec:queue`), and update `scripts/verify.sh:10` to run `ateam:doctor` and `test:spec`. Acceptance:
  `task doctor`, `task spec:check`, `task spec:queue`, and `task test:spec` exit 0.
- [x] 2.2 Slim `scripts/merge-check.sh:1-271` to delegate to `ateam review gate`, update `.github/workflows/ci.yml:25-100`
  (`fetch-depth: 0`, `ateam v0.3.8` install, `TEST_SPEC_BASE`, `required` needing `[verify]`), and add
  `.github/workflows/merge-check.yml` with base-commit checkout, `ATEAM_REQUIRED_CHECKS: Required,Merge check`, and the
  `Merge check probe` (`ateam review gate --awaiting`). Acceptance: workflow YAML parses cleanly and local
  `scripts/merge-check.sh` wrapper tests pass.

## 3. Protocol, rules, and contract tests

- [x] 3.1 Update `.agents/protocol.md:1-142`, `AGENTS.md:1-178`, `.agents/rules-index.md`, and `.agents/README.md:1-154`
  with the `ateam v0.3.8` rules (`quick`/`design` track, 60-line `proposal.md` + `Approved-by:` + `OPEN-Q` hold,
  `review_launch: "agent"` + `ateam review request`/`notify`, `ateam mutate`, `ateam test-spec --base`, Close hygiene,
  Ponytail footer + `// shortcut:` markers, and recorded divergences). Acceptance: `AGENTS.md` < 24,000 bytes, domain
  row counts match `.agents/rules-index.md`, and `task docs:check` exits 0.
- [x] 3.2 Update `internal/harness/contract/agentsetup_test.go:1-250`, replace
  `internal/harness/contract/{mergecheck,specqueue}_test.go` with `ateam` template-drift, workflow/probe/wrapper, and
  `Taskfile.yml` wiring tests plus paired sensitivity cases, and add `// spec:` citations to every new or modified test.
  Acceptance: `task verify` passes on a clean tree and `ateam mutate` detects planted harness-check mutations.
