# agent-setup

## MODIFIED Requirements

### Requirement: Thin role adapters

Each of the four roles (`semvocab-architect`, `semvocab-developer`, `semvocab-reviewer`, `semvocab-technical-writer`)
SHALL have a canonical contract under `.agents/contracts/`, a Claude adapter, and a Codex adapter. Each adapter SHALL
reference exactly its own contract, say to read it fully first, stay within 20 lines, and contain no checklist.

#### Scenario: Adapter drift is detected

- **WHEN** an adapter points at another role's contract
- **THEN** `TestAgentSetupParity` fails naming the adapter and the field

## ADDED Requirements

### Requirement: Adapter tool and model pins

Claude adapters SHALL list exactly the read-only or write tool set for the role and the role's model; Codex adapters
SHALL set the role's model and effort, with `sandbox_mode = "read-only"` on read-only roles and no sandbox override on
write roles.

#### Scenario: Adapter pin drift is detected

- **WHEN** an adapter grants a read-only role `Edit`, lists `LSP`, changes a pinned model or effort, or sets a sandbox
  on a write role
- **THEN** `TestAgentSetupParity` fails naming the adapter and the field
