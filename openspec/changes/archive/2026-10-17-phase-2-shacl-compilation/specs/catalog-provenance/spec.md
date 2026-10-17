# catalog-provenance

## ADDED Requirements

### Requirement: Pinned test fixtures

For each vendored external test fixture, `docs/admission-ledger.yaml` SHALL record, under `pins.fixtures`, the
repository, commit, file paths, sha256, and license. Tests SHALL check the sha256 before reading the fixture.

#### Scenario: The SHACL vocabulary

- **WHEN** a reader looks up the SHACL vocabulary in `pins.fixtures`
- **THEN** it names `w3c/data-shapes`, a full commit, `shacl/shacl.ttl`, its sha256, and the W3C Software and
  Document License
