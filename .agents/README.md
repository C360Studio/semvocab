# SemVocab Agent Profiles

`.agents/` holds the shared behavioral authority, workflow protocol, review policy, host manifests, and canonical skills
used across **Claude Code**, **Codex CLI**, **Gemini CLI**, and **Antigravity (`agy`)**. Platform files under
`.claude/`, `.codex/`, `.gemini/`, and `.mcp.json` are thin adapters and tool/hook registrations verified by
`task ateam:doctor` (`ateam doctor --check` at `v0.3.8`), `TestAgentSetupParity`, and `TestAteamTemplatesUnchanged`.

## Role and platform mapping

Specialist contracts under `.agents/contracts/semvocab-*.md` layer SemVocab's domain rules on top of `ateam`'s seeded
base contracts (`.agents/contracts/{architect,developer,reviewer}.md`, which remain byte-identical to `ateam v0.3.8`):

- SemVocab architect (read-only; reads `.agents/contracts/architect.md` first)
  - Canonical: `.agents/contracts/semvocab-architect.md`
  - Claude: `.claude/agents/semvocab-architect.md`
  - Codex: `.codex/agents/semvocab-architect.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-architect"`, `enable_write_tools = false`)
- SemVocab developer (reads `.agents/contracts/developer.md` first)
  - Canonical: `.agents/contracts/semvocab-developer.md`
  - Claude: `.claude/agents/semvocab-developer.md`
  - Codex: `.codex/agents/semvocab-developer.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-developer"`, `enable_write_tools = true`)
- SemVocab reviewer (independent; read-only; reads `.agents/contracts/reviewer.md` first)
  - Canonical: `.agents/contracts/semvocab-reviewer.md`
  - Claude: `.claude/agents/semvocab-reviewer.md`
  - Codex: `.codex/agents/semvocab-reviewer.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-reviewer"`, `enable_write_tools = false`)
- SemVocab technical writer
  - Canonical: `.agents/contracts/semvocab-technical-writer.md`
  - Claude: `.claude/agents/semvocab-technical-writer.md`
  - Codex: `.codex/agents/semvocab-technical-writer.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-technical-writer"`, `enable_write_tools = true`)

SemEngine's four specialist roles are carried; add another role only with a recorded owner decision.

## Model routing and review policy

Adapters separate **model tier** from **reasoning effort** (`low -> medium -> high`). Cross-agent review comparisons and
automated reviewer launches read `.agents/review-policy.json` (`"cross_provider": true`, `"review_launch": "agent"`) and
the host manifests `.agents/hosts/{enjoy,herdr}.toml`.

| Role | Claude | Codex | Gemini (Antigravity) | Target effort |
| --- | --- | --- | --- | --- |
| Architect | `opus` | `gpt-6-astra`, `medium` | `inherit` or `pro` (`enable_write_tools = false`) | `medium` default; escalate to `high` for cross-repository boundary designs |
| Developer | `opus` | `gpt-6-sol`, `medium` | `inherit` or `pro` (`enable_write_tools = true`) | `medium` default for bounded work; escalate to `high` for complex/failed-review rounds |
| Reviewer | `opus` | `gpt-6-astra`, `high` | `inherit` or `pro` (`enable_write_tools = false`) | `high` (single-pass audit of record against review-only rules) |
| Technical writer | `sonnet` | `gpt-6-sol`, `medium` | `flash` or `inherit` (`enable_write_tools = true`) | `low` or `medium` (mechanical doc and task-truth sync) |

- **Model tier:** architect, developer, and reviewer run on the frontier model. The technical writer is
  non-decision-making (doc sync, `task docs:check`, conservative task ticking) and runs on a lighter tier where
  supported.
- **Reasoning effort:** start the architect and developer at `medium`; escalate to `high` when a change crosses the
  SemEngine/SemSPARQL boundary, designs the SHACL compiler profile, or fails a review round. Keep the reviewer at
  `high`. Run the technical writer at `low` or `medium`. The tracked `.claude/agents/` and `.codex/agents/` adapters
  pin these defaults and are checked by `TestAgentSetupParity` (`internal/harness/contract/agentsetup_test.go`).
- **Cross-agent review launch:** for `implemented-by:` and `reviewed-by:`, the agent is the model family (`claude`,
  `codex`, `gemini`), not the IDE or host. Under `"review_launch": "agent"`, `ateam review request <pr>` prints eligible
  cross-provider reviewers not weaker than the implementer (`ateam review compare`) and resolves `.agents/hosts/enjoy.toml`
  or `.agents/hosts/herdr.toml` so the session can start the reviewer and receive completion via `ateam review notify`.

## Orchestrating role agents

These rules bind the session that intakes owner requests and spawns or briefs role agents:

- **Clarify Definition of Done before starting or delegating work.** Before mutating state or spawning a role agent on
  an underspecified prompt, do a quick read-only check of any named issue, PR, or file (never asking what the tree
  already answers), then verify four boundaries are clear: (1) **Deliverable**, (2) **Scope and non-goals**,
  (3) **Definition of Done and verification**, and (4) **Stop boundary**. If any is missing or ambiguous, stop, propose
  a concise 3–4 bullet recommended Definition of Done, and ask at most 1–3 targeted questions; once confirmed, run
  autonomously to that finish line.
- **A brief carries the owner's intent, not only the approved document.** State what the product is for
  (`AGENTS.md`, "What this is for") and any owner statement that bears on the question, including issue #1's ownership
  boundary with SemEngine.
- **An orchestrator's paraphrase is an unreviewed claim.** Hand an agent the artifact itself (a path, an issue comment
  URL, a `file:line`), not a summary of it. Where a summary is unavoidable, mark it as the orchestrator's so the
  reviewer checks it like any other claim.
- **Resume for continuity, start fresh for mechanics.** A resumed agent re-reads its own transcript on every resume.
  For a mechanical follow-up, a fresh agent with a precise brief costs less.
- **A design gets three review rounds.** Findings still open after the third go to the owner on the issue, with the
  choice stated, instead of a fourth round.
- **Scope an inventory before it starts.** An inventory names its question and the repositories (and commits) it reads
  first.

## Shared work protocol and rule index

`.agents/protocol.md` is the canonical shared work protocol (hybrid state homes, lifecycle rituals, cross-agent review,
worktree hygiene). `CLAUDE.md` and `AGENTS.md` point to it and carry the four gates (clarify, claim, merge, close)
inline; edit the protocol only in `.agents/protocol.md`. `.agents/rules-index.md` is the enforcement audit matrix
backing the domain-grouped rule summary in `AGENTS.md`.

## Shared skills

Canonical skills live in `skills/<name>/SKILL.md`, with matching Claude adapters in `.claude/skills/<name>/SKILL.md` and
Gemini commands in `.gemini/commands/<name>.toml`:

- [pickup](skills/pickup/SKILL.md): reconcile open issues, PRs, worktrees, review records, and untracked shortcuts via
  `ateam state pickup`.
- [handoff](skills/handoff/SKILL.md): checkpoint shared state and record the 5-field pickup prompt via `ateam state
  handoff`.
- [preflight](skills/preflight/SKILL.md): select SemVocab verification gates for the current diff and record
  reproducible evidence (`task verify`, `task test:spec`, `task mutate:check`).
- [design](skills/design/SKILL.md): run the repository-first OpenSpec `design`-track workflow (`proposal.md`,
  `design.md`, `specs/`, `tasks.md`).
- [review-pr](skills/review-pr/SKILL.md): execute an independent cross-agent PR review in a detached checkout and post
  the commit-bound `Review record` comment.
- [ponytail](skills/ponytail/SKILL.md): apply the Ponytail 6-step decision ladder, 6-line testing block, and 1–2 line
  exit summary.

All skills other than `preflight` stay byte-identical to `ateam v0.3.8` (`TestAteamTemplatesUnchanged`); `preflight` is
SemVocab-owned and carries `ateam`'s required section markers (`ateam doctor --check`).

## Read-only parity smoke

Steps 1–8 are enforced automatically by `task ateam:doctor`, `TestAgentSetupParity`, and `TestAteamTemplatesUnchanged`
(`internal/harness/contract/`). Run Step 9 after changing a contract, adapter, or repository routing rule.

1. Confirm all four `semvocab-*` contracts, three `ateam` base contracts, and eight agent adapters exist, with the
   architect, developer, and reviewer contracts pointing to their base contract first.
2. Confirm each role adapter names-and-links only its matching `.agents/contracts/semvocab-*.md` path and says to read
   it fully first.
3. Confirm Claude read-only adapters list `Read`, `Bash`, `Grep`, `Glob`, `Skill` and write adapters add `Edit` and
   `Write`, with no `Task` or `LSP`.
4. Confirm Codex read-only adapters set `sandbox_mode = "read-only"` and write adapters set no sandbox override.
5. Confirm `CLAUDE.md` only imports `@AGENTS.md`, `AGENTS.md` stays strictly under 24,000 bytes, and numbered rule
   domains and row counts match `.agents/rules-index.md`.
6. Confirm role adapters stay within 20 lines and contain no copied checklist (`- [ ]` or `- [x]`).
7. Confirm every canonical skill under `.agents/skills/` has a matching Claude adapter and Gemini `.toml` command, and
   non-owned `ateam` templates match `v0.3.8` byte-for-byte.
8. Confirm `.mcp.json`, `.agents/hooks.json`, `.claude/settings.json`, `.codex/{config.toml,hooks.json}`,
   `.gemini/settings.json`, and `.agents/review-policy.json` pass `task ateam:doctor`, and that
   `TestAteamTemplatesUnchanged` verifies `review_launch: "agent"` and `.agents/hosts/{enjoy,herdr}.toml`.
9. Validate changed skill frontmatter and relative links, then walk routing fixtures without private memory
   (`semvocab-developer` -> `semvocab-reviewer` for code, `semvocab-architect` for designs,
   `semvocab-technical-writer` for doc/task sync).
