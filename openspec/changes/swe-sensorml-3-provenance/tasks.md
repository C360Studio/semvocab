# Tasks: swe-sensorml-3-provenance

Each task names the outcome that proves it. Phase 0 received owner sign-off on PR #27 (2026-10-27); Phase 1 delivers
the catalogs, `swe-core` pack, vendored test oracles, and adopter documentation on PR #27.

## 1. Provenance and ledger schema

- [x] 1.1 `docs/admission-ledger.yaml` pins `sosa-2017`, `ssn-2017`, and `ssn-system-2017` under `pins.ontologies` and
      `swe-3.0` and `sensorml-3.0` under `pins.schemas` with 40-hex commits and 64-hex `sha256` digests
- [x] 1.2 `docs/admission-ledger.yaml` updates `sosa` and `ssn` to `new-content` and adds `swe`, `sensorml`, and
      `swe-core` with admitted-term tables, `semconnect` exclusions, and Ruling R3 namespace-honesty statements
- [x] 1.3 `docs/catalog-ownership.md` records `sosa`, `ssn`, `swe`, and `sensorml` against SemEngine `v0.1.0-alpha.2`
      (`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`) with `file:line` citations and divergence notes
- [x] 1.4 `internal/ledger` decodes `pins.schemas`, `Ontology.Created`, and typed `SOSA`/`SSN`/`SWE`/`SensorML` source
      blocks strictly, and `TestAdmissionLedgerShape` + `TestAdmissionLedgerShapeSensitivity` enforce them

## 2. Vendored oracles and catalog packages

- [x] 2.1 `internal/oracle/testdata/w3c-sdw/` and `internal/oracle/testdata/ogcapi-connected-systems/` contain the
      unmodified W3C TTLs, OGC 3.0 JSON Schemas, `LICENSE`, and `NOTICE` files; `internal/oracle` verifies every
      `sha256` against the ledger before parsing and fails when a byte is altered or a file is unlisted
- [x] 2.2 `catalog/sosa` and `catalog/ssn` export the exact ledger-admitted constants (`sosa` constants only; `ssn`
      covering `ssn` + `ssn-system` with `DirectParents`, `Parents`, and `IsA`), verified against the vendored W3C TTLs
      with paired sensitivity tests
- [x] 2.3 `catalog/swe` and `catalog/sensorml` export the exact ledger-admitted classes (`PascalCase`), properties
      (`Prop<PascalCase>`), and `SubClassOf`/`Parents`/`IsA` hierarchies (crossing from `sensorml` into `swe`),
      verified against the vendored OGC 3.0 JSON Schemas with paired sensitivity tests

## 3. Pack and adopter documentation

- [x] 3.1 `packs/swe-core.yaml`, `packs.SWECoreSpec()`, and `packs.RegisterSWECore()` load and register `sosa`, `ssn`,
      `ssn-system`, `swe`, and `sml` through SemEngine `v0.1.0-alpha.2`, compact `ssn-system` ahead of `ssn`, and leave
      registry and prefix state unchanged on failure
- [x] 3.2 `README.md`, `docs/catalogs-and-packs.md`, and `AGENTS.md` describe the new catalogs and `swe-core` pack
      accurately, with executed example output

## 4. Verification

- [x] 4.1 `task spec:check` and `task docs:check` pass
- [x] 4.2 `task verify` passes on a clean tree

## 5. Review and archive

- [ ] 5.1 `openspec archive swe-sensorml-3-provenance` moves the spec deltas to `openspec/specs/` as the last content
      commit after cross-agent review
      Hold: cross-agent review of record on PR #27 before archive
