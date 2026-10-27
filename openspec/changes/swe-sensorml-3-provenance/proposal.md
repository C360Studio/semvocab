# swe-sensorml-3-provenance

## Why

Issue #26 adds W3C SOSA/SSN (with `ssn-system`), OGC SWE Common 3.0, OGC SensorML 3.0, and the `swe-core` pack. Per
`AGENTS.md` and owner rulings R1–R3 on #26 (2026-10-27), Phase 0 records exact upstream pins, SemEngine overlap,
namespace honesty, and admitted-term tables in `docs/admission-ledger.yaml` and `docs/catalog-ownership.md` before any
catalog or pack code is written, stopping at the Phase 0 draft PR for owner sign-off.

## What Changes

- `docs/admission-ledger.yaml`:
  - Pin W3C 2017 `sosa-2017`, `ssn-2017`, and `ssn-system-2017` (`w3c/sdw` at
    `cb31ee9de636a7101337815ba67ba9af4a7c1348`) under `pins.ontologies`.
  - Add `pins.schemas` pinning OGC SWE Common 3.0 (`OGC 24-014`, edition `3.0.0`) and SensorML 3.0 (`OGC 23-000`,
    edition `3.0.0`) from `opengeospatial/ogcapi-connected-systems` `v1.0.0`
    (`8e03b236a049849f2ccc24b4fd9fdce5ff69bed2`) with per-file `sha256` digests.
  - Move `sosa` and `ssn` from `deferred` to `new-content` and add `swe`, `sensorml`, and `swe-core` entries with
    complete admitted-term tables, `semconnect` divergence exclusions, and Ruling R3 namespace-honesty statements.
- `docs/catalog-ownership.md`: update `sosa` and `ssn` rows (adding `doc.go:31`) and add `swe` and `sensorml` rows
  against SemEngine `v0.1.0-alpha.2` (`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`).
- `internal/ledger`: add `Pins.Schemas`, `Ontology.Created`, typed source decoders (`SOSA`, `SSN`, `SWE`, `SensorML`),
  shape checks, and paired sensitivity tests.
- Spec delta on `catalog-provenance`.

## Non-goals

- No Go package under `catalog/sosa`, `catalog/ssn`, `catalog/swe`, `catalog/sensorml`, or `packs/` and no vendored
  files under `internal/oracle/testdata/` (Phase 1, after Phase 0 sign-off).
- No `packs/sensorml` predicates, projections, or SHACL shapes (deferred to Phase 2 per Ruling R2).
- No claim that `http://www.opengis.net/spec/SWE/3.0/` or `http://www.opengis.net/spec/sensorML/3.0/` is an externally
  certified or dereferenceable OGC OWL ontology (Ruling R3).

## Impact

OpenSpec artifacts, `docs/admission-ledger.yaml`, `docs/catalog-ownership.md`, and `internal/ledger`. Because YAML and
Go files under `internal/ledger` change, merging Phase 0 after owner sign-off requires cross-agent review.
