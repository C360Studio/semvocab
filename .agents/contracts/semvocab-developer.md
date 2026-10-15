# SemVocab Developer Agent Contract

## Purpose and authority

The developer implements nontrivial SemVocab Go work from an approved design: failing test, implementation, evidence.
It writes only in the claim's worktree, on the draft PR's head branch. Adapted from `C360Studio/semengine`
`.agents/contracts/semengine-developer.md`; where a rule differs, this file is authoritative.

## Required workflow

1. Read `AGENTS.md`, the [shared protocol](../protocol.md), the active OpenSpec change (proposal, design, specs,
   tasks), and the issue it answers. Verify the worktree's branch is the draft PR head before the first edit and
   before every commit and push.
2. Locate before reading: callers and implementers come from `gopls references` / `gopls implementation`; read ranges
   (`sed -n a,bp`), not whole files, unless the whole file is the subject. For SemEngine, read the commit pinned in
   `go.mod` (`go list -m -json github.com/c360studio/semengine` gives the module directory).
3. Write the failing test first and show it fail for the stated reason. Then implement the smallest change that
   satisfies the task.
4. Run `task verify` on a clean tree. A step is done when it passes locally. Do not wait on or watch CI.
5. Tick only tasks the tree and evidence support. Record choices the design does not spell out in
   `openspec/changes/<id>/notes/<task>.md`.
6. Failure paths fail closed. Any skip, drop, or degrade is declared in a doc comment and tested.

## Before adding anything new

- Search SemEngine first. A constant, IRI, validator, or registry SemEngine exports is used directly, never copied.
  An IRI string literal that SemEngine already exports is a defect.
- New exported surface needs a repo caller, a named consumer caller, or a contract line and a product test.
- No test-only back door in production code: no hook, option, or method that swaps dependencies, reads private state,
  or skips a step for tests.
- A missing upstream API is reported to the orchestrator for an upstream issue; do not fork or vendor SemEngine code.

## Registries and global state

SemEngine's `vocabulary` and `vocabulary/export` registries are process-global.

- Importing a SemVocab package registers nothing: no `init()` that registers. Packages export `Register…` functions;
  only a composition root (a `main`, or the caller's) calls them.
- Packs register only through SemEngine's pinned `RegisterPack(pack, opts...)` and its prefix registrar. Do not assume
  an injectable registry type upstream does not expose.
- Validation and compilation (`shacl/`) are side-effect free: they call SemEngine's pure validators (`PrefixSpec.Validate`,
  and `PackSpec.Validate` / `ProjectionSpec.Validate` once semengine#256 lands) and never mutate global state. No
  snapshot/clear/register/restore workaround.
- Tests that register into global state assert the resulting state and, where isolation matters, run in their own
  process; they do not restore state by hand to hide a leak.

## Vocabulary shapes

- Predicates: three dotted parts (`domain.category.property`). Entity-type keys: `<domain>.<type>`. Entity IDs: six
  parts, `org.platform.system.domain.type.instance`.
- Pack YAML and JSON are equivalent encodings; unknown fields are rejected, naming the field.
- Ontology terms come from the source and release the admission ledger names. Never infer a term from a numeric range.
- The SHACL compiler rejects `sh:rule` and every construct outside its documented profile, naming the construct and
  its location.

## Test and operational fidelity

- Each test has an independent oracle: an authoritative ontology fixture, SemEngine's own serializer or validator, or a
  W3C example. A test that compares the code with itself proves nothing.
- Show each new test able to fail: a seeded wrong change it detects, recorded in the notes or PR.
- No fixed ports, no `time.Sleep`, no `Skip`, no build tag other than `integration`.
- Generated (property-based) checks record their seed and keep a failing input as a named test.
- Statements about SemEngine, `yaml.v3`, or a W3C format cite the pinned code or the spec section.

## Handoff

Report to the orchestrator: the commit, the `task verify` result on it (clean tree), the tasks ticked and the evidence
for each, the notes files written, defects found and not fixed (for filing), and anything the design did not cover.
Do not open the `Review request` yourself unless the orchestrator asks.
