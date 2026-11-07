# Proposal: admit-bfo-bridge-relations

Track: design (adds two spec Requirements, modifies `catalogs-and-packs` and `catalog-provenance`; `design.md`).

## Why

`catalog/bfo` exports ten relations carried from the SemStreams pin (`catalog/bfo/bfo.go:100-123`) and excludes the
SemStreams relation symbols whose IRIs are BFO 2.0/RO terms or source defects (`docs/admission-ledger.yaml:377-389`).
`packs/diekm/diekm.yaml:58,71,75` already steps through `BFO_0000196` (bearer of) and `BFO_0000084` (generically
depends on) with no catalog constant behind them. Ruling R1 on #31 (2026-11-06) admits a cohesive set of 15 BFO 2020
object properties in one pass, each verified against the pinned `bfo-2020` `bfo-core.ttl` fixture: type, exact label,
not deprecated, and `owl:inverseOf` / `rdfs:subPropertyOf` consistency, with paired sensitivity tests.

## What Changes

- `catalog/bfo/bfo.go`: a second relation block with the 15 R1 constants (`BearerOf`, `InheresIn`,
  `GenericallyDependsOn`, `IsCarrierOf`, `IsConcretizedBy`, `Concretizes`, `OccupiesTemporalRegion`,
  `HasOccurrentPart`, `OccurrentPartOf`, `ContinuantPartOf`, `HasContinuantPart`, `HasMemberPart`, `MemberPartOf`,
  `PrecededBy`, `Precedes`), each `Namespace + "BFO_…"`; the package comment names the additions.
- `docs/admission-ledger.yaml` `bfo` entry: `external_additions.relations` (Go name to IRI local name and label),
  `relation_axioms` (the 11 inverse pairs and 4 subproperty edges `bfo-core.ttl` asserts between admitted relations),
  and `admitted_instead` on four exclusion rows. Every exclusion stays: SemStreams' bindings remain wrong.
- `internal/ledger`: decode the new fields. `internal/oracle`: `OWLInverseOf`, `RDFSSubPropertyOf`, `IRIObjects`.
- Tests: `catalog/bfo` parity and relation-axiom checks against the fixture, with seeded-fault twins; a
  `packs/diekm` check that its BFO projection predicates are catalog relations.
- `docs/catalogs-and-packs.md`: the admitted BFO relations, their inverses, and subproperties.

## Non-goals

- No new pin: the fixture stays `bfo-2020` at sha256 `128f5fe1…a106b` (`docs/admission-ledger.yaml:43-51`).
- No exported inverse or subproperty API in `catalog/bfo` (no caller); no change to `packs/diekm/diekm.yaml`.
- No BFO 2.0 or RO IRIs (`BFO_0000050`–`053`, `RO_*`); no relations beyond R1 (`BFO_0000194`/`195`, `BFO_0000183`).
- No change to SemEngine, SemStreams, or SemSPARQL.

## Capabilities

- Modified `catalogs-and-packs`: an excluded source symbol is its name at its source IRI; new requirements for BFO
  relation declarations and relation axioms.
- Modified `catalog-provenance`: an exclusion row names the release terms admitted in its place.

## Impact

Exported surface: 15 new constants in `catalog/bfo`, no removals or value changes. Internal: `internal/ledger`,
`internal/oracle`. SemEngine stays at `v0.1.0-alpha.2`. Closes #31 (draft PR #38).
