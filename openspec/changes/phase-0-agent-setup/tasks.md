# Tasks: phase-0-agent-setup

Each task names the outcome that proves it. Command evidence is recorded on PR #2.

## 1. Agent guidance

- [x] 1.1 `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md`, `.agents/rules-index.md`, and `.agents/README.md` exist,
      adapted to SemVocab; `AGENTS.md` is under 24,000 bytes and its five domains match the rule index row for row
- [x] 1.2 Four role contracts and eight Claude/Codex adapters exist on SemEngine's model and effort routing
- [x] 1.3 Skills `semvocab-preflight`, `semvocab-handoff`, `semvocab-pickup` and their Claude adapters exist

## 2. Tooling and contract tests

- [x] 2.1 `go.mod` pins Go 1.26.9 with revive and govulncheck as tool directives; `package.json`, `.nvmrc`, and
      `.task-version` pin OpenSpec, markdownlint, Node, and Task; `task doctor` passes
- [x] 2.2 `TestAgentSetupParity`, `TestAgentSetupParitySensitivity`, `TestMergeCheck*`, and `TestSpecQueue*` pass
      under `task test:unit`
- [x] 2.3 `task verify` passes on a clean tree

## 3. CI and repository settings

- [x] 3.1 `.github/workflows/ci.yml` carries `verify`, `merge-check`, and `required`; `.github/dependabot.yml` covers
      actions, npm, and gomod
- [x] 3.2 Labels `type:epic`, `status:needs-decision`, `status:blocked`, and `class:flake` exist on
      `C360Studio/semvocab`
- [x] 3.3 A ruleset on `main` requires the `Required` check with strict status checks; `task merge:check -- 2`
      reports the up-to-date rule as present

## 4. Archive

- [ ] 4.1 `openspec archive phase-0-agent-setup` moves the specs to `openspec/specs/` as the last content commit
