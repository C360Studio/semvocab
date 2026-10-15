# SemVocab Agent Profiles

The files in `contracts/` are the tracked, platform-neutral behavioral authority. Platform adapters are intentionally
thin and must point to exactly one canonical contract. Adapted from SemEngine (`C360Studio/semengine`, `.agents/`),
which adapted SemStreams'; where a rule differs, this tree is authoritative.

## Role and platform mapping

- SemVocab architect (read-only)
  - Canonical: `.agents/contracts/semvocab-architect.md`
  - Claude: `.claude/agents/semvocab-architect.md`
  - Codex: `.codex/agents/semvocab-architect.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-architect"`, `enable_write_tools = false`)
- SemVocab developer
  - Canonical: `.agents/contracts/semvocab-developer.md`
  - Claude: `.claude/agents/semvocab-developer.md`
  - Codex: `.codex/agents/semvocab-developer.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-developer"`, `enable_write_tools = true`)
- SemVocab reviewer (independent; read-only)
  - Canonical: `.agents/contracts/semvocab-reviewer.md`
  - Claude: `.claude/agents/semvocab-reviewer.md`
  - Codex: `.codex/agents/semvocab-reviewer.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-reviewer"`, `enable_write_tools = false`)
- SemVocab technical writer
  - Canonical: `.agents/contracts/semvocab-technical-writer.md`
  - Claude: `.claude/agents/semvocab-technical-writer.md`
  - Codex: `.codex/agents/semvocab-technical-writer.toml`
  - Gemini (Antigravity): `define_subagent` (`name = "semvocab-technical-writer"`, `enable_write_tools = true`)

SemEngine's four roles are carried; SemStreams' explorer and judge contracts are not. Add one only with a recorded
owner decision.

## Model routing

Adapters separate **model tier** from **reasoning effort** (`low -> medium -> high`). Neither the architect nor the
reviewer runs on a weaker model tier than the developer.

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

On Gemini (Antigravity), role subagents are registered with `define_subagent` pointing at the matching
`.agents/contracts/semvocab-*.md` contract and invoked with `invoke_subagent`; `Model` defaults to `"inherit"` unless
`"pro"` or `"flash"` is explicitly requested. Antigravity can also host Claude models; for `implemented-by:` and
`reviewed-by:` the agent is the model family, not the tool (`.agents/protocol.md`, "Cross-agent review").

The table does not rank one platform's models against another's; across agents the pairing is the owner's decision.
For a code pull request the reviewer of record is another agent's reviewer from the table above; a documents-only pull
request keeps this repository's reviewer (`.agents/protocol.md`, "Cross-agent review"). `scripts/merge-check.sh`
recognizes `claude`, `codex`, and `gemini` on code pull requests.

## Orchestrating role agents

These rules bind the session that intakes owner requests and spawns or briefs role agents. They come from SemEngine's
account of where its orchestration cost went: every review round found something real, so the review loop is not
where to cut; the avoidable cost was in the briefs and unclarified scope.

- **Clarify Definition of Done before starting or delegating work.** Before mutating state or spawning a role agent on
  an underspecified prompt, do a quick read-only check of any named issue, PR, or file (never asking what the tree
  already answers), then verify four boundaries are clear: (1) **Deliverable** (answer/audit, issue, OpenSpec design,
  docs-only PR, or code PR), (2) **Scope and non-goals**, (3) **Definition of Done and verification**, and (4) **Stop
  boundary**. If any is missing or ambiguous, stop, propose a concise 3–4 bullet recommended Definition of Done, and
  ask at most 1–3 targeted questions; once confirmed, run autonomously to that finish line.
- **A brief carries the owner's intent, not only the approved document.** State what the product is for
  (`AGENTS.md`, "What this is for") and any owner statement that bears on the question, including issue #1's
  ownership boundary with SemEngine.
- **An orchestrator's paraphrase is an unreviewed claim.** Hand an agent the artifact itself (a path, an issue
  comment URL, a `file:line`), not a summary of it. Where a summary is unavoidable, mark it as the orchestrator's so
  the reviewer checks it like any other claim.
- **Resume for continuity, start fresh for mechanics.** A resumed agent re-reads its own transcript on every resume.
  For a mechanical follow-up, a fresh agent with a precise brief costs less.
- **A design gets three review rounds.** Findings still open after the third go to the owner on the issue, with the
  choice stated, instead of a fourth round.
- **Scope an inventory before it starts.** An inventory names its question and the repositories (and commits) it
  reads first.

## Shared work protocol and rule index

`.agents/protocol.md` is the canonical shared work protocol (state homes, rituals, worktree hygiene). `CLAUDE.md` and
`AGENTS.md` point to it and carry the four gates (clarify, claim, merge, close) inline; edit the protocol only in
`.agents/protocol.md`. `.agents/rules-index.md` is the enforcement audit matrix backing the domain-grouped rule
summary in `AGENTS.md`.

## Shared skills

Canonical skills live in `skills/`. Read the relevant `SKILL.md` fully; the platform adapters add discovery metadata
and argument handling, not another rule set. Repository protocol and role contracts remain authoritative.

- [semvocab-preflight](skills/semvocab-preflight/SKILL.md): scope existing verification gates and record their
  evidence. Claude command `/preflight`.
- [semvocab-handoff](skills/semvocab-handoff/SKILL.md): publish shared state and preserve unfinished work. Claude
  command `/semvocab-handoff`.
- [semvocab-pickup](skills/semvocab-pickup/SKILL.md): reconcile current state and verify worktree ownership. Claude
  command `/semvocab-pickup`.

SemEngine's engine-capability skills (`entity-or-bucket`, `kv-or-stream`, `new-payload`, `query-pattern`) govern NATS
storage and graph code SemVocab does not own, and are not carried. Add a capability skill here when a SemVocab
capability (catalog admission, pack authoring, SHACL profile) has settled guidance worth one.

Codex can invoke the canonical names with `$`, or read their paths directly. Antigravity (Gemini) automatically
discovers `.agents/skills/<name>/SKILL.md` from the workspace root. Claude uses the command names above; its adapters
live in `.claude/skills/<command-name>/SKILL.md`. The `preflight` adapter deliberately maps to the canonical
`semvocab-preflight` name. A private memory reference cannot be required to interpret a shared rule.

## Read-only parity smoke

Steps 1–8 are enforced automatically by `TestAgentSetupParity` and `TestAgentSetupParitySensitivity`
(`internal/harness/contract/agentsetup_test.go`, run by `task test:unit`). Run Step 9 after changing a contract,
adapter, or repository routing rule.

1. Confirm all four canonical contracts and all eight agent adapters exist, and that `.agents/protocol.md` and
   `.agents/rules-index.md` exist.
2. Confirm each adapter names-and-links only its matching `.agents/contracts/...` path and says to read it fully first.
3. Confirm the Claude reviewer and architect tool lists contain `Read`, `Bash`, `Grep`, `Glob`, `Skill`, and that
   write adapters add `Edit` and `Write`, with no `Task`, `LSP`, or other delegation tool (`gopls` is reached through
   `Bash`; an unresolved tool name makes Claude Code refuse to launch the agent).
4. Confirm the Codex reviewer and architect set `sandbox_mode = "read-only"`; the developer and the technical writer
   have no sandbox override and inherit the parent workspace permissions.
5. Confirm `CLAUDE.md` only imports `AGENTS.md` and carries no rules of its own, that `AGENTS.md` stays strictly under
   24,000 bytes, and that every rule row in `AGENTS.md` has its corresponding row in `.agents/rules-index.md` (the test
   checks numbered domain headings and per-domain row counts; semantic correspondence is review only).
6. Confirm role adapters stay within 20 lines and contain no copied checklist (`- [ ]` or `- [x]`).
7. For every skill in the shared-skills list, confirm the canonical file and Claude adapter exist, the adapter points
   to that canonical file and says to read it fully, and it contains no copied checklist.
8. Confirm the Claude and Codex model and effort settings match the table above.
9. Validate changed skill frontmatter and relative links, then walk the applicable scenarios below without private
   memory.

Use these semantic fixtures when reading the routing text:

- "Implement a nontrivial change" routes first to the SemVocab developer and then the SemVocab reviewer.
- "Review a nontrivial SemVocab change" routes to the SemVocab reviewer in read-only mode.
- "Design a catalog, pack, or SHACL compiler target" routes to the SemVocab architect (ownership inventory first,
  drafts as text); binding rulings and approval stay with the owner session.
- "Update durable docs or reconcile task truth" routes to the SemVocab technical writer.
- "Check an isolated Go idiom" may use a generic Go agent only as a second pass.
- "Prepare a docs-only PR" retains protocol review/CI gates and selects checks for the changed artifacts.
- "The previous revision was green" preserves that evidence as historical, not current-head proof.
- "Continue without private memory" resolves shared state and verifies ownership through pickup/protocol.

The smoke passes only when Claude, Codex, and Gemini resolve the same logical role and canonical contract for every
fixture.
