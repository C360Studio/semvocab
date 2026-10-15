# spec-requirement-length

## Why

OpenSpec 1.14.1 (Dependabot PR #3) warns when a requirement's text runs over 500 characters, and
`openspec validate --strict` treats that warning as a failure. Two requirements on `main` are over the limit:
agent-setup "Thin role adapters" (596 characters) and catalog-provenance "One disposition per candidate" (526). Until
they are shorter, `task spec:check` fails on #3. The owner chose to fix the specs rather than stay on 1.14.0
(2026-10-15, PR #3).

## What Changes

- agent-setup: "Thin role adapters" keeps the contract and adapter-shape rules. The tool, model, effort, and sandbox
  rules move, word for word, into a new requirement, "Adapter tool and model pins".
- catalog-provenance: "One disposition per candidate" keeps the candidate list and the disposition vocabulary. The
  per-disposition field rules move, word for word, into a new requirement, "Disposition fields".
- Every SHALL statement is unchanged; only the requirement it sits under changes. Each existing scenario keeps its
  name under its original requirement. Each new requirement gets a scenario that `TestAgentSetupParity` or
  `TestAdmissionLedgerShape` already enforces.

## Impact

Specs only; no code, test, or ledger changes. After this merges, Dependabot can rebase #3 and `task spec:check`
passes on 1.14.1.
