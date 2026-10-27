# Tasks: SensorML 3.0 and SOSA/SSN Vocabulary Pack

## 1. Phase 2a: Ledger disambiguation, provenance, and design

- [x] 1.1 Update internal/ledger with Kind, ID, Catalog, Pack, and fail-closed Entry, updating callers in oracle and
      tests
- [x] 1.2 Add pack: sensorml to docs/admission-ledger.yaml with 40 verified terms, 4 local assertions, and ruling notes
- [x] 1.3 Verify task spec:check, task docs:check, and task verify pass on the Phase 2a change branch

## 2. Phase 2b: Pack implementation, SHACL verification, and docs

- [x] 2.1 Implement packs/sensorml (sensorml.yaml, sensorml.shacl.ttl, sensorml.mapping.yaml, sensorml.go) exporting
      Spec, Register, Shapes, and Mapping.
- [x] 2.2 Add unit, JSON and YAML equivalence, SHACL compilation, projection contract, and candidate-state validation
      tests in packs/sensorml/sensorml_test.go and shacl/.
- [x] 2.3 Extend packs/packs_test.go and packs/registration_test.go to verify packs/sensorml against oracle loaders,
      local assertions closures, SemEngine constants, and swe-core coexistence.
- [x] 2.4 Update README.md, docs/catalogs-and-packs.md, and AGENTS.md for packs/sensorml and run verification gates.
- [ ] 2.5 Archive sensorml-pack into openspec/specs/ after cross-agent review.
