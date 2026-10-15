# agent-setup

## Purpose

The agent setup is the tracked guidance every coding agent (Claude, Codex, Gemini) loads in this repository: one
`AGENTS.md`, canonical role contracts with thin per-platform adapters, shared skills, and a rule index. A test holds
its structure so platforms resolve the same role and contract.

## ADDED Requirements

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

Each of the four roles (`semvocab-architect`, `semvocab-developer`, `semvocab-reviewer`, `semvocab-technical-writer`)
SHALL have a canonical contract under `.agents/contracts/`, a Claude adapter, and a Codex adapter. Each adapter SHALL
reference exactly its own contract, say to read it fully first, stay within 20 lines, and contain no checklist. Claude
adapters SHALL list exactly the read-only or write tool set for the role and the role's model; Codex adapters SHALL set
the role's model and effort, with `sandbox_mode = "read-only"` on read-only roles and no sandbox override on write
roles.

#### Scenario: Adapter drift is detected

- **WHEN** an adapter points at another role's contract, grants a read-only role `Edit`, lists `LSP`, changes a pinned
  model or effort, or sets a sandbox on a write role
- **THEN** `TestAgentSetupParity` fails naming the adapter and the field

### Requirement: Skill adapters

Every directory under `.agents/skills/` SHALL have a Claude adapter (`semvocab-preflight` maps to `preflight`) that
links the canonical `SKILL.md`, says to read it fully, stays within 25 lines, and contains no checklist.

#### Scenario: A new skill without an adapter

- **WHEN** a canonical skill is added with no Claude adapter
- **THEN** `TestAgentSetupParity` fails naming the missing adapter path
