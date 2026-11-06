# agent-setup

## MODIFIED Requirements

### Requirement: Thin role adapters

Each of the four roles (`semvocab-architect`, `semvocab-developer`, `semvocab-reviewer`, `semvocab-technical-writer`)
SHALL have a canonical contract under `.agents/contracts/`, a Claude adapter, and a Codex adapter. The architect,
developer, and reviewer contracts SHALL instruct reading their `ateam` base contract
(`.agents/contracts/{architect,developer,reviewer}.md`) first. Each adapter SHALL reference only its `semvocab-*`
contract, say to read it fully first, stay within 20 lines, and contain no checklist.

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

## ADDED Requirements

### Requirement: Ateam bootstrap and doctor parity

`task ateam:doctor` SHALL run `ateam doctor --check` at the pinned `v0.3.8` release with session environment variables
cleared, verifying `.mcp.json`, `.agents/hooks.json`, `.agents/review-policy.json` (`review_launch` set to `"agent"`),
`.agents/hosts/{enjoy,herdr}.toml`, base contracts, canonical skills, and client settings without mutating the worktree.
`TestAteamTemplatesUnchanged` SHALL fail when a seeded `ateam` template file drifts from `v0.3.8`.

#### Scenario: Seeded template drifts from the pin

- **WHEN** `.agents/contracts/reviewer.md` or `.agents/skills/review-pr/SKILL.md` is edited away from the `v0.3.8`
  template
- **THEN** `ateam doctor --check` and `TestAteamTemplatesUnchanged` fail naming the file

### Requirement: Changed-test spec citations

`task test:spec` SHALL run `ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"`, requiring every new or modified Go
`Test`, `Fuzz`, or `Example` since the merge base to carry a valid `// spec: openspec/specs/<cap>/spec.md#Requirement:
<name>` comment and failing on any broken citation anywhere in the repository while leaving untouched pre-adoption tests
exempt.

#### Scenario: New test omits a spec citation

- **WHEN** a branch adds or edits a Go test function without a `// spec:` comment
- **THEN** `task test:spec` fails naming the uncited test
