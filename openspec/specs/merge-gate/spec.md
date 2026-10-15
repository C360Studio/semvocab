# merge-gate Specification

## Purpose
The merge gate is what must hold before a pull request merges to `main`: the same verification locally and in CI, a
known-flake stop, an up-to-date head, and cross-agent review lines on code pull requests.

## Requirements

### Requirement: One verification entrypoint

`task verify` SHALL run `spec:check`, `docs:check`, `fmt:check`, `tidy:check`, `build`, `vet`, `lint`, `vuln`,
`test:unit`, and `test:repeat` in that order, continue past a failing step to report every broken gate, print each
step's time, and fail if any step failed or verification changed a tracked file. CI's `verify` job SHALL run
`task verify` on `ubuntu-24.04` with Go from `go.mod`, Node from `.nvmrc`, and Task from `.task-version`.

#### Scenario: Verification edits a tracked file

- **WHEN** a step rewrites a tracked file during `task verify`
- **THEN** `task verify` fails with `tree-unchanged` and lists the changed file

### Requirement: Known-flake check

`scripts/merge-check.sh` SHALL fail while an open issue labelled `class:flake` exists that the pull request's
description does not close.

#### Scenario: An open flake blocks an unrelated pull request

- **WHEN** a `class:flake` issue is open and the pull request does not declare `Closes #n` for it
- **THEN** `merge-check` fails naming the issue (`TestMergeCheckKnownFlake`)

### Requirement: Up-to-date rule

`scripts/merge-check.sh` SHALL fail unless the rules on `main` require a branch to be up to date before merging.

#### Scenario: Ruleset lacks the up-to-date setting

- **WHEN** `main`'s rules do not set strict required status checks
- **THEN** `merge-check` fails saying so (`TestMergeCheckUpToDateRule`)

### Requirement: Cross-agent review lines

On a code pull request that is not a draft, `scripts/merge-check.sh` SHALL require exactly one `implemented-by:` line
naming at least one of `claude`, `codex`, `gemini` and exactly one `reviewed-by:` line naming exactly one agent that
`implemented-by:` does not name (any agent when it names more than one or the author is a bot). A pull request whose
changed files are all Markdown or under `openspec/`, with none under `.claude/agents/` or `.agents/contracts/`, is
exempt.

#### Scenario: Self-review on a code pull request

- **WHEN** a non-draft pull request changes a script and both lines name `claude`
- **THEN** `merge-check` fails (`TestMergeCheckReview`)

### Requirement: Single required check

CI's `required` job SHALL depend on `verify` and `merge-check`, always run, and fail unless every needed job
succeeded. The `main` ruleset SHALL require the `Required` check with strict (up-to-date) status checks.

#### Scenario: A needed job is skipped

- **WHEN** `merge-check` is cancelled or skipped
- **THEN** `Required` fails
