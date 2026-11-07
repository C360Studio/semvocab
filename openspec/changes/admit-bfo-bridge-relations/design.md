# Design: admit-bfo-bridge-relations

## Context

Ruling R1 on #31 (2026-11-06) admits 15 BFO 2020 object properties into `catalog/bfo`. Inventory at `01db18a`:

- `catalog/bfo/bfo.go:100-123` exports 10 relations, all ported from SemStreams `8b99efe9` and named in the ledger's
  `symbols_admitted_relations` (`docs/admission-ledger.yaml:372-373`). `TestCatalogPortParity`
  (`catalog/bfo/bfo_test.go:150-160`) requires each admitted name to equal the SemStreams fixture value
  (`catalog/bfo/testdata/semstreams-bfo.txt`), so names absent from or wrong in SemStreams cannot join that list.
- External additions are BFO 2020 classes only: `external_additions.terms` (`docs/admission-ledger.yaml:375`) is
  checked as `owl:Class` with its label (`catalog/bfo/bfo_test.go:177-181`).
- The exclusion check (`catalog/bfo/bfo_test.go:144-148`) fails if any excluded name is exported. R1 reuses three
  excluded names at new IRIs (`BearerOf` `BFO_0000053`→`196`, `InheresIn` `052`→`197`, `OccupiesTemporalRegion`
  `132`→`199`; rows at `docs/admission-ledger.yaml:380-389`) and admits two excluded IRIs under new names (`PrecededBy`
  = `BFO_0000062`, `OccurrentPartOf` = `BFO_0000132`). Exclusion by name alone or by IRI alone breaks R1.
- `internal/oracle/oracle.go:144-153` reads objects only through unexported `objects`; no exported accessor returns
  IRI objects, so `owl:inverseOf` and `rdfs:subPropertyOf` are unreadable from tests today.
- SemEngine `v0.1.0-alpha.2` (`go.mod:12`) has no BFO IRI in any non-test Go file (grep of the module
  cache, 2026-11-07): disposition stays new content from `bfo-2020`, no re-export.
- Callers: `packs/diekm/diekm.yaml:58,71,75` uses `BFO_0000196` and `BFO_0000084` as literals; `diekm_test.go:195-204`
  restates them as literals. No Go caller of any R1 constant exists; R1 is the scope authority.

Fixture facts (`internal/oracle/testdata/bfo-2020/bfo-core.ttl`, sha256 `128f5fe1…a106b`, 0 `owl:deprecated`):

| Go name | IRI | Label line | Declared axioms (line) |
| --- | --- | --- | --- |
| `BearerOf` | `BFO_0000196` | 541 | inverseOf `197` (531); subPropertyOf `194` (530) |
| `InheresIn` | `BFO_0000197` | 558 | subPropertyOf `195` (548) |
| `GenericallyDependsOn` | `BFO_0000084` | 250 | inverseOf `101` (240) |
| `IsCarrierOf` | `BFO_0000101` | 267 | none |
| `IsConcretizedBy` | `BFO_0000058` | 175 | inverseOf `059` (167) |
| `Concretizes` | `BFO_0000059` | 189 | none |
| `OccupiesTemporalRegion` | `BFO_0000199` | 573 | `owl:FunctionalProperty` (565); no inverse |
| `HasOccurrentPart` | `BFO_0000117` | 301 | inverseOf `132` (296); transitive |
| `OccurrentPartOf` | `BFO_0000132` | 370 | transitive |
| `ContinuantPartOf` | `BFO_0000176` | 426 | inverseOf `178` (422) |
| `HasContinuantPart` | `BFO_0000178` | 437 | none |
| `HasMemberPart` | `BFO_0000115` | 289 | inverseOf `129` (285); subPropertyOf `178` (284) |
| `MemberPartOf` | `BFO_0000129` | 359 | subPropertyOf `176` (355) |
| `PrecededBy` | `BFO_0000062` | 201 | inverseOf `063` (196); transitive |
| `Precedes` | `BFO_0000063` | 212 | transitive |

Every subject line is `rdf:type owl:ObjectProperty` with exactly one `rdfs:label`, equal to R1's. Already-admitted
relations gain two edges: `HasTemporalPart` (`121`) subPropertyOf `117` (308) and `TemporalPartOf` (`139`)
subPropertyOf `132` (377). Inverse pairs among the 25 admitted relations: 11 (the 7 above plus `054`/`055` line 103,
`056`/`057` 124, `124`/`171` 321, `121`/`139` 309). `OccursIn`'s inverse `BFO_0000183` (221) stays outside the set.

## Goals / Non-Goals

Goals: the 15 constants, each proven against the fixture; ledger rows that say where each came from and what they
replace; inverse and subproperty records checked both ways. Non-goals: an exported axiom API, new pins, `diekm.yaml`.

## Decisions

**D1. Placement and names.** (a) A second `const` block after `bfo.go:123`, headed "Relations added from BFO 2020
(ruling R1 on #31)", R1 order and names, comment style `// Name (BFO_x, "label") …` as at `:102-122`. (b) Merge into
the existing block: hides which relations have SemStreams provenance. Choose (a). Adopter seam: callers import
`bfo.BearerOf` etc.; nothing existing changes value.

**D2. Ledger record of the additions.** (a) `external_additions.relations: {GoName: [local name, label]}`, the CCO
shape (`docs/admission-ledger.yaml:407-408`), checked as `owl:ObjectProperty` + exact label + not deprecated. (b) Put
them in `symbols_admitted_relations` and skip SemStreams parity for absent names: silently drops parity for the
three reused names. Choose (a).

**D3. Exclusions.** Rows stay: SemStreams' bindings remain wrong. (a) Optional `admitted_instead: ["<GoName>
<BFO_x>"]` per row; test rule: an excluded name is exported only if its row's `admitted_instead` names it, at that
IRI, and that entry is in `external_additions.relations`; no exported name carries its excluded IRI. Rows:
`PartOf`/`HasPart` → `ContinuantPartOf BFO_0000176`, `HasContinuantPart BFO_0000178`; `BearerOf`/`InheresIn` →
`BearerOf BFO_0000196`, `InheresIn BFO_0000197`; `PrecedesTemporally` → `Precedes BFO_0000063`, `PrecededBy
BFO_0000062`; `OccupiesTemporalRegion` → `OccupiesTemporalRegion BFO_0000199`. (b) Implicit: allow any excluded
name the ledger admits elsewhere: cheaper, but a typo silently re-admits a name. (c) Delete rows: loses the defect
record. Choose (a). `Entry.Excluded()` is unchanged, so the five other catalogs are untouched.

**D4. Inverse and subproperty consistency.** (a) Ledger `relation_axioms` (`inverse_pairs`, `sub_property_of`)
must equal, both ways, the fixture's axioms between admitted relations; plus each R1 relation's declared inverse
must be admitted. (b) Exported `bfo.InverseOf`/`SubPropertyOf`: new surface with no caller (AGENTS.md § 4). (c)
Test-local table: same proof, but `docs/catalogs-and-packs.md` would then cite no checked record. Choose (a).
Requires `oracle.IRIObjects`, `OWLInverseOf`, `RDFSSubPropertyOf`.

**D5. Tests.** Oracle: the fixture, read through `internal/oracle` after its sha256 check. Each check has a
seeded-fault twin (tasks 3.2-3.4). The fixture has no deprecated term, so the not-deprecated check is proved against
`oracle.LoadMerged(BFO2020, CCO22)` with a CCO v2.2 deprecated IRI. `task mutate:check` mutants: (1) `BearerOf` →
`BFO_0000053`; (2) `PrecededBy`/`Precedes` IRIs swapped; (3) `OccupiesTemporalRegion` → `BFO_0000132`; (4) the
either-direction match in the inverse comparison reduced to one direction; (5) `IRIObjects` keeping literals.

**D6. `packs/diekm`.** (a) Leave `diekm.yaml` literal (YAML cannot name a Go constant) and add one test binding its
OBO-namespace projection predicates to `bfo.BearerOf`/`bfo.GenericallyDependsOn` (#31's scope lists
`diekm_test.go`). (b) No pack change: the catalog would stay unused in-repo. The existing literal oracle at
`diekm_test.go:195-204` stays independent. Choose (a). Deferred: generating pack YAML from constants; checking other
packs' predicates.

## Risks / Trade-offs

- `relation_axioms` must be edited whenever a relation is admitted; the both-ways check makes a missed edit fail.
- Declared cost: `admitted_instead` adds one optional field to the shared `Exclusion` type.

## Migration Plan

Additive: no constant is removed or changes value. Rollback is reverting the PR.

## Open Questions

1. (Owner) D3: the spec now defines an excluded symbol as a name at its source IRI, so R1's `BearerOf`, `InheresIn`,
   and `OccupiesTemporalRegion` reuse excluded names. Confirm `admitted_instead` (a) over (b).
2. (Owner) D4: relation axioms recorded in the ledger, with no exported Go API. Confirm (a).
3. (Owner) D6: the `diekm` binding test is in scope. Confirm (a), or drop task 3.4 and its scenario.

Architect: claude (claude-opus-5-5, high).
