# Tasks: swe-sensorml-3-provenance

Each task names the outcome that proves it. Phase 0 stops at the draft PR for owner sign-off (ruling R2 on #26).

## 1. Provenance and ledger schema

- [x] 1.1 `docs/admission-ledger.yaml` pins `sosa-2017`, `ssn-2017`, and `ssn-system-2017` under `pins.ontologies` and
      `swe-3.0` and `sensorml-3.0` under `pins.schemas` with 40-hex commits and 64-hex `sha256` digests
- [x] 1.2 `docs/admission-ledger.yaml` updates `sosa` and `ssn` to `new-content` and adds `swe`, `sensorml`, and
      `swe-core` with admitted-term tables, `semconnect` exclusions, and Ruling R3 namespace-honesty statements
- [x] 1.3 `docs/catalog-ownership.md` records `sosa`, `ssn`, `swe`, and `sensorml` against SemEngine `v0.1.0-alpha.2`
      (`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`) with `file:line` citations and divergence notes
- [x] 1.4 `internal/ledger` decodes `pins.schemas`, `Ontology.Created`, and typed `SOSA`/`SSN`/`SWE`/`SensorML` source
      blocks strictly, and `TestAdmissionLedgerShape` + `TestAdmissionLedgerShapeSensitivity` enforce them

## 2. Verification

- [x] 2.1 `task spec:check` and `task docs:check` pass
- [x] 2.2 `task verify` passes on a clean tree

## 3. Sign-off and archive

- [ ] 3.1 `openspec archive swe-sensorml-3-provenance` moves the spec delta to `openspec/specs/` as the last content
      commit after owner sign-off and cross-agent review
      Hold: owner sign-off on Phase 0 draft PR (ruling R2 on #26) and cross-agent review before archive
