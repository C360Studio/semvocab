# agent-setup Specification

## Purpose
The agent setup is the tracked guidance every coding agent (Claude, Codex, Gemini) loads in this repository: one
`AGENTS.md`, canonical role contracts with thin per-platform adapters, shared skills, and a rule index. A test holds
its structure so platforms resolve the same role and contract.

## Requirements

### Requirement: One guidance file

`AGENTS.md` SHALL be the single guidance file, strictly under 24,000 bytes. `CLAUDE.md` SHALL import it with
`@AGENTS.md`, stay within 10 lines, and carry no section heading or rule table of its own.

#### Scenario: Oversized or forked guidance is refused

- **WHEN** `AGENTS.md` reaches 24,000 bytes, or `CLAUDE.md` drops the import or adds a `##` heading
- **THEN** `TestAgentSetupParity` fails naming the file

### Requirement: Rule index parity

`AGENTS.md` and `.agents/rules-index.md` SHALL carry the same five numbered rule domains, in order, with the same
number of rows in each.

#### Scenario: A rule added to one file only

- **WHEN** a rule row is removed from one domain of `AGENTS.md` but not from `.agents/rules-index.md`
- **THEN** `TestAgentSetupParity` fails naming both files and the domain

### Requirement: Thin role adapters

Each role (`semvocab-architect`, `semvocab-developer`, `semvocab-reviewer`, `semvocab-technical-writer`) SHALL have a
`.agents/contracts/` contract plus Claude and Codex adapters. The architect, developer, and reviewer contracts SHALL
instruct reading `.agents/contracts/{architect,developer,reviewer}.md` first. Each adapter SHALL link only its
`semvocab-*` contract, say to read it fully first, stay within 20 lines, and contain no checklist.

#### Scenario: Adapter drift is detected

- **WHEN** an adapter points at another role's contract or a specialist contract drops its base-contract link
- **THEN** `TestAgentSetupParity` fails naming the file and missing reference

### Requirement: Skill adapters

Every directory under `.agents/skills/` (`pickup`, `handoff`, `preflight`, `design`, `review-pr`, `ponytail`) SHALL have
a matching Claude adapter under `.claude/skills/<name>/SKILL.md` and Gemini command under `.gemini/commands/<name>.toml`.
Each Claude adapter SHALL link its canonical `SKILL.md`, say to read it fully, stay within 25 lines, and contain no
checklist.

#### Scenario: A new skill without an adapter

- **WHEN** a canonical skill is added without a matching Claude or Gemini adapter
- **THEN** `TestAgentSetupParity` or `ateam doctor --check` fails naming the missing adapter path

### Requirement: Adapter tool and model pins

Claude adapters SHALL list exactly the read-only or write tool set for the role and the role's model; Codex adapters
SHALL set the role's model and effort, with `sandbox_mode = "read-only"` on read-only roles and no sandbox override on
write roles.

#### Scenario: Adapter pin drift is detected

- **WHEN** an adapter grants a read-only role `Edit`, lists `LSP`, changes a pinned model or effort, or sets a sandbox
  on a write role
- **THEN** `TestAgentSetupParity` fails naming the adapter and the field

### Requirement: Ateam bootstrap and doctor parity

`task ateam:doctor` SHALL run `ateam doctor --check` at the pinned `v0.3.8` release with session environment variables
cleared, verifying `.mcp.json`, `.agents/hooks.json`, `.agents/review-policy.json`, base contracts, canonical skills,
and client settings without mutating the worktree. `TestAteamTemplatesUnchanged` SHALL fail when a seeded `ateam`
template drifts from `v0.3.8`, `review_launch` is not `"agent"`, or `.agents/hosts/{enjoy,herdr}.toml` is missing or
malformed.

#### Scenario: Seeded template drifts from the pin

- **WHEN** `.agents/contracts/reviewer.md` or `.agents/skills/review-pr/SKILL.md` is edited away from the `v0.3.8`
  template
- **THEN** `ateam doctor --check` or `TestAteamTemplatesUnchanged` fails naming the file

### Requirement: Changed-test spec citations

`task test:spec` SHALL run `ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"`, requiring every new or modified Go
`Test`, `Fuzz`, or `Example` since the merge base to carry a valid `// spec: openspec/specs/<cap>/spec.md#Requirement:
<name>` comment and failing on any broken citation anywhere in the repository while leaving untouched pre-adoption tests
exempt.

#### Scenario: New test omits a spec citation

- **WHEN** a branch adds or edits a Go test function without a `// spec:` comment
- **THEN** `task test:spec` fails naming the uncited test
