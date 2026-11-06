# Proposal: SensorML 3.0 and SOSA/SSN Vocabulary Pack

## Why

Phase 1 of issue #26 (PR #27) delivered pinned provenance and catalogs (`catalog/sosa`, `catalog/ssn`, `catalog/swe`,
`catalog/sensorml`) plus `packs/swe-core`. Consumers led by `C360Studio/semconnect` still maintain local SensorML and
OGC API — Connected Systems Part 1 predicate registrations, projection contracts, and three IRI defects
(`sosa:hasLocation` on `sensorml.process.position`, `sosa:isHostedBy` path collision on `sensorml.process.attached-to`,
and `ssn:hasProperty` path collision on `sensorml.capability.value`). SemVocab needs `packs/sensorml` shipping the
canonical `PackSpec`, SHACL shapes, and projection contract mapping for the five Part 1 resources.

## What Changes

- Disambiguate `catalog` and `pack` lookups in `internal/ledger` (`Kind()`, `ID()`, `Catalog(name)`, `Pack(name)`, with
  `Entry(name)` failing closed on ambiguous names like `sensorml`) and update existing test and oracle callers.
- Add `pack: sensorml` (`disposition: new-content`, `phase: 2`) to `docs/admission-ledger.yaml` with 40
  `referenced_terms_verified` CURIEs, 4 SensorML-to-SOSA/SSN `local_assertions` (ruling R5 on #26), and explicit
  exclusion of Working-Draft OGC CS API Part 2 `csapi:` terms (ruling R4 on #26).
- Add `packs/sensorml` (`sensorml.yaml`, `sensorml.shacl.ttl`, `sensorml.mapping.yaml`, `sensorml.go`) exporting
  `Spec() (vocabulary.PackSpec, error)`, `Register(opts ...vocabulary.PackOption) error`, `Shapes() []byte`, and
  `Mapping() []byte` (matching `packs/diekm`, so callers pass `Spec()`, `shacl.ParseTurtle`, and
  `shacl.DecodeMappingYAML` to `shacl.Compile`).
- Declare 9 prefixes (`rdf`, `rdfs`, `dc`, `skos`, `sosa`, `ssn`, `ssn-system`, `swe`, `sml`), 5 `csapi.*` entity
  types, 20 `sensorml.*` predicates (including `sosa:hosts`/`sosa:isHostedBy` inverses and 1-step `ProjectionSpec`
  chains on `sensorml.capability.value` and `sensorml.characteristic.value`), 5 open SHACL node shapes, and 5 reconcile
  projection contracts.
- Add multi-oracle, registration (`packs.RegisterBFOCCOCore`, `packs.RegisterSWECore`, `diekm.Register`,
  `sensorml.Register`), SHACL compilation, and candidate-state validation tests, and update `README.md`,
  `docs/catalogs-and-packs.md`, and `AGENTS.md`.

## Capabilities

### Modified Capabilities

- `catalog-provenance`: Disambiguates catalog versus pack lookups when names match, and records explicit cross-standard
  `rdfs:subClassOf` `local_assertions` for pack closures.
- `catalogs-and-packs`: Generalizes pack superclass closures to include pack `local_assertions` and specifies the
  SensorML 3.0 and SOSA/SSN vocabulary pack and SHACL bundle.

## Impact

- Modified Go files: `internal/ledger/ledger.go`, `internal/ledger/ledger_test.go`, `internal/oracle/schema.go`,
  `catalog/sensorml/sensorml_test.go`, `packs/packs_test.go`, `packs/registration_test.go`, and `shacl/` tests.
- New Go and asset files: `packs/sensorml/sensorml.go`, `packs/sensorml/sensorml.yaml`,
  `packs/sensorml/sensorml.shacl.ttl`, `packs/sensorml/sensorml.mapping.yaml`, `packs/sensorml/sensorml_test.go`, and
  `packs/sensorml/testdata/`.
- Modified documentation and ledger files: `docs/admission-ledger.yaml`, `README.md`, `docs/catalogs-and-packs.md`,
  `AGENTS.md`.
- Dependencies and licensing: SemEngine stays pinned to `v0.1.0-alpha.2`; SemVocab files are MIT-licensed.
