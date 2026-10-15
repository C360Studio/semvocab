# Design: phase-1-catalogs

## Context

The Phase 0 design fixed releases (D1–D3), admission rules (D4), closure and IRI rules (D5, D6), the DIEKM namespace
(D7), and re-registration semantics (D8). This design records what Phase 1 decides on top: the SemEngine pin move and
the four proposed defaults Q6–Q9, which are implementer choices, not owner rulings.

## Decisions

### P1. SemEngine pin moves to `v0.1.0-alpha.2`

`v0.1.0-alpha.2` = `11da4721c2233a8e2b5b1d6236a97a44e06abbb6` contains alpha.1 plus semengine#261 (Stage 1.1:
`(*PackSpec).Validate`, `ProjectionSpec.Validate`, `export.SubjectIRI`, blank-node/IRI-predicate export; closes
semengine#256) and #263 (semengine#259 wildcard prefix query). Phase 1 uses only APIs present at alpha.1, so the move
adds no requirement; it lets Phase 2 start without another pin change. The ledger's `pins.semengine` records alpha.2,
and `phase_gates.phase-2` becomes `available` because semengine#256 is merged and released in the pinned tag. Line
numbers the Phase 0 design cites at alpha.1 move (for example `RegisterPack` is `vocabulary/pack.go:237` at alpha.2).

### P2. `bfo-cco-core` carries prefixes only (Q6)

The pack declares the `obo` and `cco` prefixes and nothing else. No consumer needs registered BFO/CCO entity types,
and registering ontology classes as entity types would claim instance data shapes no one has asked for.

### P3. `TestAdmissionLedgerShape` lands in Phase 1 (Q7)

A test decodes `docs/admission-ledger.yaml` and checks the shape the `catalog-provenance` spec requires (one entry per
candidate, one disposition each, reasons on deferrals, sources and oracles on admitted entries, pins with full
revisions). The catalog tests read their admitted-term lists from the ledger, so the ledger is the single table.

### P4. Oracle form: vendored release TTLs plus a test-only Turtle reader (Q8)

The CCO v2.2 merged TTL and BFO core TTL are vendored under `testdata` at their recorded sha256, with the BSD-3-Clause
and CC-BY-4.0 notices. A test-only reader in `internal/` parses the Turtle subset these files use and fails loudly on
anything it does not support. It adds no third-party module. The tests check the sha256 before trusting the file.

### P5. Prefix `obo` for `http://purl.obolibrary.org/obo/` (Q9)

The namespace is the OBO Foundry PURL base, shared by every OBO ontology. Naming it `bfo` would compact non-BFO OBO
terms as `bfo:…` in serializer output. The packs therefore declare `obo`, and the issue #1 fixture's `bfo` prefix
becomes `obo`. The Go package stays `catalog/bfo`, which names the ontology, not the prefix. The owner confirmed this
as ruling Q9 (#1, 2026-10-15).

### P6. DIEKM altitude predicate is `intel.measurement.altitude-m` (Q11, owner ruling)

The issue #1 fixture's `intel.measurement.altitude_m` fails SemEngine's predicate segment grammar `[a-z][a-z0-9-]*`
(`vocabulary/predicate_contract.go:144-171`, at alpha.1 and alpha.2), so `RegisterPack` rejects the pack. Owner ruling
Q11 (#1 comment 6201463524) names it `intel.measurement.altitude-m`. Its IRI `diekm:altitudeMeters` and `units: m`
are unchanged. SemSPARQL#1 references to the predicate use the new name (reported there, not edited here).

## Declared costs

- About 2.1 MB of vendored Turtle in the repository (Phase 0 declared cost).
- The test-only reader supports a subset of Turtle; a future oracle in another syntax needs reader work.
