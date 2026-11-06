# merge-gate Specification

## Purpose
The merge gate is what must hold before a pull request merges to `main`: the same verification locally and in CI, a
known-flake stop, an up-to-date head, and cross-agent review lines on code pull requests.

## Requirements

### Requirement: One verification entrypoint

`task verify` SHALL run `ateam:doctor`, `spec:check`, `test:spec`, `docs:check`, `fmt:check`, `tidy:check`, `build`,
`vet`, `lint`, `vuln`, `test:unit`, and `test:repeat` in that order, continue past a failing step, print timings, and
fail on step failure or tracked-file drift. CI's `verify` job SHALL run on `ubuntu-24.04` with `fetch-depth: 0`, Go from
`go.mod`, Node from `.nvmrc`, Task from `.task-version`, and `ateam v0.3.8`, running `task doctor` then `task verify`.

#### Scenario: Verification edits a tracked file

- **WHEN** a step rewrites a tracked file during `task verify`
- **THEN** `task verify` fails with `tree-unchanged` and lists the changed file

### Requirement: Known-flake check

`scripts/merge-check.sh` SHALL delegate to `ateam review gate` at the pinned release, which fails while an open issue
labelled `class:flake` exists that the pull request's description does not close.

#### Scenario: An open flake blocks an unrelated pull request

- **WHEN** a `class:flake` issue is open and the pull request does not declare `Closes #n` for it
- **THEN** `scripts/merge-check.sh` fails via `ateam review gate` naming the issue

### Requirement: Up-to-date rule

`scripts/merge-check.sh` SHALL delegate to `ateam review gate` and `.github/workflows/merge-check.yml` SHALL set
`ATEAM_REQUIRED_CHECKS` to `Required,Merge check`, failing unless `main`'s active ruleset requires every listed check
(`Required` when unset locally) on an up-to-date branch.

#### Scenario: Ruleset lacks the up-to-date setting

- **WHEN** `main`'s active ruleset omits a required check in `ATEAM_REQUIRED_CHECKS` or disables strict status checks
- **THEN** `scripts/merge-check.sh` fails via `ateam review gate`

### Requirement: Cross-agent review lines

On a non-draft code pull request, `scripts/merge-check.sh` SHALL invoke `ateam review gate` to require valid
`implemented-by:` and `reviewed-by:` lines plus an approving commit-bound `Review record` comment from a non-weaker
cross-provider reviewer under `.agents/review-policy.json`. Documents-only pull requests remain exempt from
cross-provider enforcement.

#### Scenario: Self-review on a code pull request

- **WHEN** a non-draft code pull request names the same provider on both lines or lacks a head-bound `Review record`
- **THEN** `scripts/merge-check.sh` fails via `ateam review gate`

### Requirement: Required CI and Merge check workflows

CI's `required` job (`Required`) SHALL depend on `verify`, always run, and fail unless `verify` succeeded.
`.github/workflows/merge-check.yml` SHALL check out `github.event.pull_request.base.sha`, install `ateam v0.3.8`, run
`ateam review gate --awaiting` in `awaiting`, skip and rename `merge-check` on drafts or when `awaiting=true`, and
otherwise run `scripts/merge-check.sh` as `Merge check`. `TestCIWorkflowPinned` SHALL guard both workflows and pins.

#### Scenario: Ready code pull request awaits its first review record

- **WHEN** a non-draft code pull request has no `reviewed-by:` line and no `Review record` comment
- **THEN** `awaiting` outputs `awaiting=true` and `merge-check` is skipped under its awaiting name so no passing
  `Merge check` status exists

#### Scenario: Probe fails or pull request has a review record

- **WHEN** `awaiting` outputs `awaiting=false` or exits without an output on a non-draft pull request
- **THEN** `merge-check` runs `scripts/merge-check.sh` under the required check name `Merge check`

### Requirement: OpenSpec queue and check wiring

`task spec:check` SHALL run `npx --no-install openspec validate --all --strict --no-interactive` followed by
`ateam state spec-check`, and `task spec:queue` SHALL run `ateam state spec-queue`. `TestSpecTaskWiring` SHALL fail if
either task drops a command, swaps command order, or sets `ignore_error`.

#### Scenario: Misplaced hold or broken spec wiring

- **WHEN** a `Hold:` appears outside every task in `tasks.md` or `Taskfile.yml` alters `spec:check` command order
- **THEN** `task spec:check` or `TestSpecTaskWiring` fails naming the violation
