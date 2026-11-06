# Shared work protocol (Claude, Codex, and Gemini)

Canonical. `CLAUDE.md` and `AGENTS.md` point to this file and carry the four gates (clarify, claim, merge, close)
inline; edit the protocol here only. Read it before taking, landing, or closing work; the `pickup`, `handoff`,
`preflight`, `design`, `review-pr`, and `ponytail` skills read it. Adapted from `ateam v0.3.8` and SemEngine's
`.agents/protocol.md`; where a rule differs, this file is authoritative.

## 1. Hybrid State Architecture

State that all agents must see lives in the repository's tools and forge, with local session coordination cached in
`.ateam/state.sqlite` via `ateam state` and the `ateam` MCP server (`ateam_state_pickup`, `ateam_state_handoff`,
`ateam_task`). There is no `/tickets` prose state.

- **What is wanted, what kind, is it decided:** a GitHub issue with labels `type:epic`, `status:needs-decision`,
  `status:blocked`. `status:needs-decision` is the owner's docket; a ruling is posted as an issue comment and the
  label removed. `status:blocked` names its blocker in a comment.
- **An epic:** a tracking issue labeled `type:epic` whose body carries a task list of `#n` children. Issue #1 is
  phased (Phases 0–3 required, Phase 4 deferred); a PR delivering one phase says `Addresses #1` and names the phase.
- **Who has claimed what:** a **draft PR** on an agent-prefixed branch (`claude/<topic>`, `codex/<topic>`,
  `gemini/<topic>`) in its own worktree (`git worktree list`), with `Closes #n` for a leaf issue or `Addresses #n` for
  an epic or phase. No draft PR, no claim.
- **Target state, tracks, task truth, holds:** the OpenSpec change under `openspec/changes/<id>/`; `task spec:queue`
  (`ateam state spec-queue`) shows its holds and `task spec:check` (`openspec validate --strict` + `ateam state
  spec-check`) validates structure:
  - **Tracks:** `quick` (`proposal.md` + `tasks.md` + delta `specs/` when behavior changes) for bounded single-capability
    edits with no new spec Requirement; `design` (adds `design.md`, drafted via `.agents/skills/design/SKILL.md`) when
    adding a spec Requirement, crossing a SemEngine/SemSPARQL seam, or touching multiple capabilities.
  - **Proposal & approval:** `proposal.md` stays within 60 lines and carries `Approved-by: <name> (<YYYY-MM-DD>)` once
    the owner approves. Until approved, `tasks.md` carries `Hold: OPEN-Q proposal awaits owner approval` inside task
    `1.1`.
  - **Holds & tasks:** every task carries its `file:line` premise and acceptance check, and can be ticked in or before
    the archive commit. No task may assert a post-merge fact ("CI green", "merge-ready"). A hold is written inside the
    unticked task it stops as `Hold: <reason>`; a `Hold:` outside every task fails `task spec:check`.
- **Session handoff & shortcuts:** `ateam state pickup` reconciles open issues, PRs, worktrees, review records, and
  untracked `// shortcut:` markers; `ateam state handoff` records the 5-field pickup prompt at session boundaries.
- **Upstream dependencies:** an issue on the owning repository (SemEngine for the vocabulary API, SemSPARQL for
  projection expansion), linked from the SemVocab issue or task it blocks, with that task carrying the `Hold:`.
- **Why:** an ADR (`docs/adr/`), or the owner's ruling comment on the issue.

## Bounded port scope

A port (catalog material carried or extracted from the SemStreams pin) implements the admitted content and its accepted
contract. A finding adds required work only when the correction satisfies a binding requirement (issue #1's acceptance
criteria, a spec requirement), an admission or verification gate, an explicit owner ruling, an accepted design
commitment, or the dependency change needed to build. Name that obligation and the concrete failure or missing
evidence. A maintenance preference alone (inherited duplication, naming, indirection) is optional: it cannot veto
approval or become `BLOCKING`/`HIGH`, and scheduling it needs an explicit owner decision. This never excuses a broken
contract, a newly introduced defect, a copied SemEngine constant, or an unproved admission gate.

## 2. Work Lifecycle & Definition of Done

1. **Pickup & Clarify:** run `ateam state pickup` (or `/pickup`) to inspect open issues, draft PR claims, worktrees, and
   `task spec:queue`. Before claiming work, mutating state, or spawning role subagents, read any named issue, PR, or
   file so you never ask a question the repository already answers, then verify four boundaries are clear:
   (1) **Deliverable** (answer/audit, issue, OpenSpec design, docs-only PR, or code PR), (2) **Scope and non-goals**
   (in-bounds packages/files and what must not change), (3) **Definition of Done and verification** (observable
   completion criteria and `task` gates), and (4) **Stop boundary** (pause after inventory/design for an owner decision
   vs run through to ready-for-review or merge). When a prompt leaves any of the four ambiguous, stop first, propose a
   concise 3–4 bullet recommended Definition of Done, and ask at most 1–3 targeted questions; once set, run
   autonomously to that stop boundary.
2. **Claim:** create or switch to an isolated worktree at `.worktrees/<agent>-<topic>` (the gitignored directory inside
   the repository) unless the host creates it for you (Enjoy's `move_to_worktree`, herdr), on an agent-prefixed branch
   (`claude/…`, `codex/…`, `gemini/…`), push, and open a draft PR with `Closes #n` or `Addresses #n`. The primary
   checkout is discovery-only. Immediately before every commit and push, verify `git rev-parse --abbrev-ref HEAD`
   matches the draft PR head. Update a pushed branch by merging `origin/main`, never by rebase or force-push.
3. **File:** any defect found in repository code that this change does not fix is filed as a GitHub issue (`gh issue
   list --search` first; `status:blocked` when the code exists only on an open PR branch). Declare any intentional code
   shortcut with a `// shortcut: <reason> (#<issue>)` comment; untracked `// shortcut:` markers surface in `ateam state
   pickup`. Sister repositories are strictly read-only.
4. **Verify:** a step is done when `task verify` passes locally on a clean tree with `$(go env GOPATH)/bin` first on
   `PATH` (`ateam:doctor` parity green, `spec:check` green, `test:spec` citations green on changed tests, `docs:check`,
   `fmt:check`, `tidy:check`, `build`, `vet`, `lint`, `vuln`, `test:unit`, and `test:repeat` green), non-trivial
   refusals/branches are checked with `task mutate:check` (`ateam mutate`), developer task choices not in `design.md`
   are recorded in `openspec/changes/<id>/notes/<task>.md`, and every conclusion states Ponytail's 1–2 line exit
   summary (`Skipped:` and `Residual risk:`).
5. **Cross-agent review:** on a non-draft code pull request (any file outside `.md` and `openspec/**/*.{md,yml,yaml,json}`,
   or any file under `.claude/agents/` or `.agents/contracts/`), `scripts/merge-check.sh` (`ateam review gate`)
   enforces `.agents/review-policy.json`:
   - The PR body carries one `implemented-by: <agent> (<model>, <effort>)` line and, once approved, one
     `reviewed-by: <agent> (<model>, <effort>)` line naming a cross-provider reviewer (`claude`, `codex`, `gemini` — the
     model family, not the IDE) that is not weaker by model tier then effort band (`ateam review compare`).
     Documents-only PRs and dependency-bot PRs accept any agent's reviewer.
   - Because `.agents/review-policy.json` sets `"review_launch": "agent"`, when ready for review run
     `ateam review request <pr>` (passing `--return-to <conversation-id>` in Enjoy), follow the host launch instruction
     from `.agents/hosts/{enjoy,herdr}.toml` (or post a `Review request` PR comment when no host applies), and record a
     handoff if stopping.
   - The reviewer checks out a detached review worktree (`.worktrees/<agent>-review-pr-<n>`), audits the tree and runs
     its own checks, posts a PR comment starting `reviewed-by: <agent> (<model>, <effort>)` with `Reviewed at <sha>` and
     `Verdict: APPROVE` or `Verdict: REQUEST CHANGES` per `.agents/contracts/reviewer.md` § Record, updates the PR
     description's `reviewed-by:` line on approval, removes the review checkout, and notifies the implementer via
     `ateam review notify`. A later commit that is not a clean `main` merge or docs-only requires a new review record.
6. **Handoff or Land (Close):** at a session boundary or before context compaction, run `ateam state handoff` (or
   `/handoff`) to record the 5-field pickup prompt. When a PR completes an OpenSpec change, move it to
   `openspec/changes/archive/` (`npx --no-install openspec archive <id> --yes`, syncing `openspec/specs/`) as a commit
   on the branch **before** marking the PR ready (`Hold: OPEN-Q proposal awaits owner approval` must already be gone).
   Once approved, `Required` and `Merge check` are green on a head up to date with `main`, no `class:flake` issue is
   open, and `task merge:check -- <n>` passes locally (`GITHUB_ACTIONS` unset), arm squash auto-merge (`gh pr merge
   --squash --auto`). After the merge lands, remove the claimed worktree and remote branch (`ateam state prune`).

## 3. Known Flakes & Recorded Divergences

- **Known flakes:** an open issue labelled `class:flake` (passes and fails on the same tree) blocks every PR in
  `Merge check` unless the PR closes every open `class:flake` issue and documents reproduction before/after.
- **Recorded SemVocab divergences from `ateam`'s stock template:** (1) `ateam bench --selftest` is omitted because
  SemVocab owns ontology catalogs/packs/SHACL rather than `ateam`'s agent-benchmark harness; (2) `task test:spec` runs
  `ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"` so pre-adoption tests remain untouched while every new or
  edited test requires `// spec:`; (3) ontology port `design.md` files may exceed 120 lines (`ateam state spec-check`
  warns non-fatally) to hold mandatory symbol-by-symbol provenance inventories; (4) Docker/network harness scripts are
  omitted because SemVocab runs no networked containers. Gate selection lives in [preflight](skills/preflight/SKILL.md).
