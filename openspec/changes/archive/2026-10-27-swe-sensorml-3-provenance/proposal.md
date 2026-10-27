# swe-sensorml-3-provenance

## Why

Issue #26 adds W3C SOSA/SSN (with `ssn-system`), OGC SWE Common 3.0, OGC SensorML 3.0, and the `swe-core` pack. Per
`AGENTS.md` and owner rulings R1–R3 on #26 (2026-10-27), Phase 0 recorded exact upstream pins, SemEngine overlap,
namespace honesty, and admitted-term tables in `docs/admission-ledger.yaml` and `docs/catalog-ownership.md` and
received owner sign-off on PR #27 (`7ef6dc9de85507a0d6cfcc229796df0d8565e580`); Phase 1 delivers the vendored test
oracles, the `catalog/{sosa,ssn,swe,sensorml}` packages, the `packs/swe-core` pack, and adopter documentation.

## What Changes

- `docs/admission-ledger.yaml`, `docs/catalog-ownership.md`, and `internal/ledger` (Phase 0, signed off):
  - Pin W3C 2017 `sosa-2017`, `ssn-2017`, and `ssn-system-2017` (`w3c/sdw` at
    `cb31ee9de636a7101337815ba67ba9af4a7c1348`) under `pins.ontologies`, and OGC SWE Common 3.0 (`OGC 24-014` `3.0.0`)
    and SensorML 3.0 (`OGC 23-000` `3.0.0`) from `opengeospatial/ogcapi-connected-systems` `v1.0.0`
    (`8e03b236a049849f2ccc24b4fd9fdce5ff69bed2`) under `pins.schemas` with per-file `sha256` digests.
  - Record complete admitted-term tables, `semconnect` divergence exclusions, Ruling R3 namespace-honesty statements,
    and strict `internal/ledger` decoders and sensitivity tests.
- `internal/oracle` and `internal/oracle/testdata/` (Phase 1):
  - Vendor unmodified `w3c/sdw` TTLs (`internal/oracle/testdata/w3c-sdw/`) and `opengeospatial/ogcapi-connected-systems`
    JSON Schemas + `LICENSE` (`internal/oracle/testdata/ogcapi-connected-systems/`) with `NOTICE` files, verifying
    every file's `sha256` against `docs/admission-ledger.yaml` before parsing.
- `catalog/sosa`, `catalog/ssn`, `catalog/swe`, and `catalog/sensorml` (Phase 1):
  - Export exact ledger-admitted constants and hierarchies, verified against the vendored W3C TTLs and OGC JSON Schemas
    plus paired sensitivity tests.
- `packs/swe-core.yaml`, `packs.SWECoreSpec()`, and `packs.RegisterSWECore()` (Phase 1):
  - Embedded prefix pack binding `sosa`, `ssn`, `ssn-system`, `swe`, and `sml`, with SemEngine `v0.1.0-alpha.2`
    registration, `ssn-system` compaction, and atomic failure tests.
- `README.md`, `docs/catalogs-and-packs.md`, `AGENTS.md`, and spec deltas on `catalog-provenance` and
  `catalogs-and-packs`.

## Non-goals

- No `packs/sensorml` predicates, projections, or SHACL shapes (deferred to Phase 2 per Ruling R2).
- No claim that `http://www.opengis.net/spec/SWE/3.0/` or `http://www.opengis.net/spec/sensorML/3.0/` is an externally
  certified or dereferenceable OGC OWL ontology (Ruling R3).
- No mutation of sister repositories (`semengine`, `semstreams`, `semsparql`, `semconnect`).

## Impact

New public Go packages (`catalog/sosa`, `catalog/ssn`, `catalog/swe`, `catalog/sensorml`), new exports in `packs`
(`SWECoreSpec`, `RegisterSWECore`), vendored test fixtures under `internal/oracle/testdata/`, and documentation/spec
updates. Requires cross-agent review of record before archive and merge.
