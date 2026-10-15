# Shared work protocol (Claude, Codex, and Gemini)

Canonical. `CLAUDE.md` and `AGENTS.md` point to this file and carry the four gates (clarify, claim, merge, close)
inline; edit the protocol here only. Read it before taking, landing, or closing work; the pickup and handoff skills
read it. Carried from SemEngine's `.agents/protocol.md`; SemEngine's owner rulings that shaped it are cited there, and
apply here only as this file restates them.

State that all agents must see lives in the repository's tools, never in a prose document or an agent's private
memory. Each question has one home, and each home is a `gh` or `task` query. There is no `/tickets` state.

- **What is wanted, what kind, is it decided:** a GitHub issue with labels `type:epic`, `status:needs-decision`,
  `status:blocked`. `status:needs-decision` is the owner's docket; a ruling is posted as an issue comment and the
  label removed. `status:blocked` names its blocker in a comment.
- **An epic:** a tracking issue labeled `type:epic` whose body carries a task list of `#n` children. Issue #1 is
  phased (Phases 0–3 required, Phase 4 deferred); a PR delivering one phase says `Addresses #1` and names the phase.
- **Who has claimed what:** a **draft PR** on an agent-prefixed branch (`claude/…`, `codex/…`, `gemini/…`) opened at the
  start of the work, with `Closes #n` for a leaf issue or `Addresses #n` for an epic or phase. No draft PR, no claim.
  Design-phase work claims the same way; the OpenSpec change is its first content commit. A stop point goes in the PR
  description.
- **Target state, task truth, holds:** the OpenSpec change inside that PR; `task spec:queue`, run in the claim's
  worktree, shows its holds. The archive (`openspec archive <id>` + spec sync) is the landing PR's last commit,
  reviewed with the code. Every task can be ticked in or before the archive commit. No task may assert a post-merge
  fact ("CI green", "merge-ready") or wait on a step that follows the archive (the check of the archive, undraft, the
  final CI run, the merge): such a task strands the change, and those steps are recorded on the PR. A hold is written
  in the unticked task it stops, on any of its lines, as `Hold:` followed by what it waits for (an issue or PR number,
  or the owner's ruling). A hold in a heading or a paragraph outside every task does not show, and `task spec:check`
  fails on it.
- **Upstream dependencies:** an issue on the owning repository (SemEngine for the vocabulary API, SemSPARQL for
  projection expansion), linked from the SemVocab issue or task it blocks, with that task carrying the `Hold:`.
- **Why:** an ADR, or the owner's ruling comment on the issue.

## Bounded port scope

A port (catalog material carried or extracted from the SemStreams pin) implements the admitted content and its
accepted contract. A finding adds required work only when the correction satisfies a binding requirement (issue #1's
acceptance criteria, a spec requirement), an admission or verification gate, an explicit owner ruling, an accepted
design commitment, or the dependency change needed to build. Name that obligation and the concrete failure or missing
evidence. A maintenance preference alone (inherited duplication, naming, indirection) is optional: it cannot veto
approval or become `BLOCKING`/`HIGH`, and scheduling it needs an explicit owner decision. This never excuses a broken
contract, a newly introduced defect, a copied SemEngine constant, or an unproved admission gate.

## Work lifecycle

- **Start:** `gh issue list --state open` · `gh pr list` (drafts are claims; skip them) · `task spec:queue` ·
  `gh run list --branch main --limit 3` · `gh issue list --label status:needs-decision`.
- **Clarify:** before claiming work, mutating state, or spawning role subagents, read any named issue, PR, or file so
  you never ask a question the repository already answers, then verify four boundaries are clear: (1) **Deliverable**
  (answer/audit, issue, OpenSpec design, docs-only PR, or code PR), (2) **Scope and non-goals** (in-bounds
  packages/files and what must not change), (3) **Definition of Done and verification** (observable completion criteria
  and `task` gates), and (4) **Stop boundary** (pause after inventory/design for an owner decision vs run through to a
  ready PR). When a prompt leaves any of the four ambiguous, stop first, propose a concise 3–4 bullet recommended
  Definition of Done the owner can approve or adjust in one reply, and ask at most 1–3 targeted questions; once set,
  run autonomously to that stop boundary without pausing on routine steps.
- **Take work:** an unclaimed issue, then a dedicated worktree on an agent-prefixed branch (`claude/<topic>`,
  `codex/<topic>`, `gemini/<topic>`; the prefix names who claimed it), then push, then a draft PR with `Closes #n` or
  `Addresses #n`, then work. One claimed PR owns one worktree. When multiple agents share a host, the primary checkout
  is discovery-only and no agent commits from it. Immediately before every commit and push, verify that the worktree's
  current branch is the draft PR head; a mismatch stops the operation. The prefix says who claimed the work, never
  whether that agent's worktree is idle.
- **Bootstrap exception:** while the repository has no base protocol, the Phase 0 bootstrap (claim PR #2, branch
  `phase-0-bootstrap`, unprefixed) is claimed under a bounded exception, as SemEngine's SETUP 01 was. The exception
  ends when that PR merges; every later claim follows the rules above.
- **Worktree hygiene:** for standalone CLI sessions, the claim's worktree lives at a durable sibling path
  (`git worktree add ../semvocab-wt/<branch> -b <branch> origin/main`), never under `/private/tmp`, which a reboot
  purges; Enjoy desktop sessions use `move_to_worktree`, which creates `.worktrees/<branch>` (ignored in `.gitignore`
  alongside `.enjoy-logs/`). Run `git worktree remove` when the PR merges. Heavy local gates run one agent at a time on
  a shared host. CI is the arbiter; a local red under contention is not a finding.
- **File:** a defect in code in this repository that the change does not fix is always filed as an issue, whether the
  code was carried, extracted, or written here. A defect is a behaviour that breaks what the code promises: a spec
  requirement, a package `README.md`, an admission-ledger contract field, a doc comment, or a reasonable reading of the
  code. Check for an existing issue first (`gh issue list --search`). When the defect's code exists only on a pull
  request's branch, the issue is labelled `status:blocked` with that pull request named as its blocker, and the pull
  request body links it with no closing keyword (`Follow-up after this pull request merges: #n`). A defect in a sister
  repository is reported to its owner, not fixed here.

  Route every other finding before opening an issue. A residual (a cost of a decision this change made deliberately)
  belongs in a doc comment at the line or in the change's `design.md`. An unmeasured cost becomes a *Declared cost*
  section in the design. Only an architectural finding (it crosses files, would need its evidence re-collected, or
  changes what someone should not do next) becomes an issue. Ask the owner when placement is a scheduling call.
- **Land:** implementation review (another agent's for a code pull request: "Cross-agent review" below), then fixes and
  re-review, then archive as the final content commit, then a narrow reviewer check of the archive/spec sync (any
  agent's reviewer), then undraft, then CI green on a head that is up to date with `main` and with **no known flake
  open**, then `task merge:check -- <n>` immediately before merging, in a shell where `GITHUB_ACTIONS` is not set, then
  squash merge. A green run while a known flake is open is not a fix: fix the flake. A correction after archive
  re-enters reconciliation and final review. State `implemented-by:` in the PR body, naming the implementing agent with
  the word `claude`, `codex`, or `gemini` beside the model or persona (`implemented-by: codex (gpt-6-sol
  semvocab-developer)`). Bring a pushed branch up to date by merging `origin/main` into it; do not rebase or force-push
  it: a review record names the commit it read. Developer notes on choices a task made that its design does not spell
  out go in `openspec/changes/<id>/notes/<task>.md`, linked from a one-line PR comment; they archive with the change.
- **Cross-agent review:** on a code pull request the implementation review, and the re-review of a fix that changes
  code, are done by an agent that wrote none of the commits under review, as the pull request's `implemented-by:` line
  records. It works in every direction: Codex's `semvocab-reviewer` reviews what Claude or Gemini sessions implement,
  and so on. A **documents-only** pull request changes only Markdown files or files under `openspec/`, and none under
  `.claude/agents/` or `.agents/contracts/`; it keeps any agent's reviewer. Any other changed file (a `.go` file,
  `go.mod`, a script, `Taskfile.yml`, the CI workflow, a pack YAML or Turtle file, `.claude/agents/*`,
  `.codex/agents/*`, `.agents/contracts/*`) makes it a code pull request. If multiple agents wrote commits, the owner
  names the reviewer on the issue. A pull request no agent wrote (a dependency bot's) is reviewed by any agent's
  reviewer. An agent does not start another platform's agent, so the pull request carries both halves:
  - The implementer asks with a PR comment headed `Review request`: the kind of review, the commit to read, the diff
    range, the implementer's `task verify` result on that commit (clean tree), and the issue or ruling the change
    answers. It does not say what the reviewer should conclude.
  - The reviewing session answers with a PR comment headed `Review record`: its reviewer's report as written, in the
    reviewer contract's format. It names the commit read, what was run, and what could not be run. Evidence comes from
    the tree and the reviewer's own runs; without its own `task verify`, the implementer's result is recorded as
    reported, never as passed. The record does not rest on CI: the CI run before merge is the check.
  - A record covers the commit it names. A later commit that changes code goes back to the reviewing agent; a later
    documents-only commit and the archive check take any agent's reviewer. A merge of `origin/main` that touches no
    file the pull request changes carries the record over.
  - The reviewer does not write on the branch. If the agents disagree on a finding, it goes to the owner on the issue,
    labelled `status:needs-decision`.
  - The implementer writes one `reviewed-by:` line in the PR body, beside `implemented-by:`, once the reviewing
    agent's newest record approves, naming that agent with `claude`, `codex`, or `gemini`.
  - The check: `scripts/merge-check.sh` (CI `merge-check`, `task merge:check -- <n>`) requires, on a code pull request
    that is not a draft, exactly one line starting `implemented-by:` naming at least one agent and exactly one line
    starting `reviewed-by:` naming exactly one agent that `implemented-by:` does not name (any agent when it names
    more than one, or the author is a bot). On a draft the findings print and fail nothing. Editing the description
    starts no CI run, so re-run the failed `merge-check` job after the edit. Whether the lines are true, and the
    record's existence and verdict, stay review only: the owner and the agents share one GitHub login.
- **Known flakes:** a known flake is an open issue labelled `class:flake`: a test or check that passes and fails on the
  same tree. A network fetch that did not answer is not one; record it on the pull request and re-run. A red you cannot
  explain by your diff is filed before the next push. While a known flake is open, `merge-check` fails on every pull
  request except one whose description closes every open flake. A pull request that closes a flake shows the
  reproduction: the command, and how often it failed before and after. A flake that comes back reopens its issue.
- **Close:** the squash merge of a PR that declared `Closes #n` at review time closes that issue; the merge is the
  authorization. A PR that only `Addresses` an epic closes nothing. A close with no merged PR behind it takes the
  owner's word on the issue itself; approval of adjacent work never widens into a close. A `class:flake` issue is
  closed, or its label removed, only by a merged fix or on the owner's word.

## Verification

CI has three jobs. `verify` runs `task verify`, the same commands as a local run. `merge-check` runs
`scripts/merge-check.sh`: it fails while a known flake is open that the pull request does not close, fails unless the
rules on `main` require an up-to-date head, and fails a code pull request that is not a draft unless its
`implemented-by:` and `reviewed-by:` lines name the agents as "Cross-agent review" says. `required` fails if `verify`
or `merge-check` failed, is missing, or was skipped or cancelled; it is the check the ruleset on `main` requires. Gate
selection and evidence recording are in [semvocab-preflight](skills/semvocab-preflight/SKILL.md).

There is no program baton document. Agents mutate only this repository.
