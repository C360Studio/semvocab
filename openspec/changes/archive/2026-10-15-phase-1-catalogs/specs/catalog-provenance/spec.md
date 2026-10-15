# catalog-provenance

## MODIFIED Requirements

### Requirement: Pinned sources

`docs/admission-ledger.yaml` SHALL record the full 40-hex SemStreams revision. It SHALL record the SemEngine module
version together with the commit it resolves to. For each external ontology it SHALL record repository, tag, commit,
file path, sha256, version IRI, namespace, and license.

#### Scenario: Reading the SemEngine pin

- **WHEN** a reader looks up `pins.semengine` in the ledger
- **THEN** it names `v0.1.0-alpha.2` and `11da4721c2233a8e2b5b1d6236a97a44e06abbb6`, and lists semengine#256 as
  included

### Requirement: Phase gates

The ledger SHALL record each issue #1 phase's dependency gate and its status at the time of the last ledger change.

#### Scenario: Phase 2 gate

- **WHEN** a reader looks up `phase_gates.phase-2`
- **THEN** it names semengine#256 merged and tagged, with status `available`
