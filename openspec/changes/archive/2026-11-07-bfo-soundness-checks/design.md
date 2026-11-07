# Design: bfo-soundness-checks

## Context

Inventory at `9a192f4` (main `e3fecf2`):

- SemEngine `v0.1.0-alpha.2` (`go.mod:12`): `PredicateSpec` has no subject class (`vocabulary/pack.go:81-96`);
  `Range` is free-form text (`vocabulary/predicates.go:282-287`); step 0 is traversed "from the entity", later steps
  from the previous node (`vocabulary/projection.go:13-15`). No non-test SemEngine Go file names a BFO IRI.
- Pack selection: `ledger.Entry.References` (`internal/ledger/ledger.go:146`) is `[bfo-2020, cco-2.2]` for `diekm`
  (`docs/admission-ledger.yaml:1075`) and `bfo-cco-core` (`:1104`), and names no BFO/CCO pin for `swe-core` or
  `sensorml`. The embedded packs are listed at `packs/packs_test.go:62-67`.
- Closures: `TestPackSuperclassClosure` (`packs_test.go:387`) already makes each `superclasses` list equal the
  release closure. `cco.IsA` (`catalog/cco/hierarchy.go:78-88`, falling back to bfo at `:54-59`) classifies every
  class diekm uses (`catalog/cco/cco.go:45-89`; `catalog/bfo/hierarchy.go:7-40`).
- Subjects: `diekm.mapping.yaml:21-29` binds both predicates to `intel.platform`; `shacl.DecodeMappingYAML`
  (`shacl/decode.go:99`), `ContractBinding` (`shacl/model.go:104-117`).
- Oracle: `objects` is unexported (`internal/oracle/oracle.go:146-155`); `IRIObjects` skips blank nodes
  (`:171-181`); collections are `rdf:first`/`rest` on `_:bN` blank nodes (`internal/turtle/turtle.go:401-472`).
- `bfo-core.ttl` disjointness: `:686-692`, `:814-816`, `:825-827`, `:1018-1032`, `:1040-1055`, and five
  `owl:AllDisjointClasses` at `:1185-1223`. Domain and range of all 17 relations in D4 at `:102-573`.
- The seven buckets are issue #32's grouping (after `jupyter-naas/abi`), not a BFO 2020 term. They cover every
  `bfo-2020` class except `BFO_0000141` itself and the continuant fiat boundaries `140/142/146/147`
  (`bfo-core.ttl:1063-1128`).
- Existing #31 binding: `packs/diekm/diekm_test.go:247-298`; unchanged.

## Goals / Non-Goals

Goals: the four checks, each proven by a seeded fault; diekm and bfo-cco-core pass. Non-goals: see `proposal.md`.

## Decisions

**D1. Placement.** (a) One new file `packs/bfo_soundness_test.go` in `packs_test`, reusing `embedded` and
`ledger.Load`. (b) A new `internal/bfoalign` package called from pack tests: a reusable seam, but it adds a package
with no non-test caller and duplicates `embedded`. Choose (a). Adopter seam: a new BFO pack is checked by being
added to `embedded` and the ledger, as `TestPackSuperclassClosure` already requires.

**D2. Selection.** (a) The ledger entry's `references` contains `bfo-2020` or `cco-2.2`. (b) The pack declares the
`obo` or `cco` prefix: the same today, but not a recorded fact. Choose (a). A scenario pins the selected set to
`{bfo-cco-core, diekm}`, so a selector that selects nothing fails.

**D3. Step-0 subject.** (a) Each entity type whose mapping contract lists the predicate (birth or group). A projected
predicate whose step 0 is a D4 relation and that has no bound subject is reported, so the check fails closed.
(b) Every entity type sharing the predicate's domain segment: diekm fails, because `intel.observation` (a Process)
becomes a bearer of a Role. (c) Skip the step-0 domain: misses a Process bearing a Role. (d) A test-local table:
duplicates the mapping. Choose (a). This forecloses BFO packs without a SHACL mapping (Open Question 2).

**D4. Domain and range source.** (a) Read each relation's `rdfs:domain`/`rdfs:range` from `bfo-core.ttl` at test
time, interpreting three shapes: a named class, `owl:unionOf` of named classes, and `owl:intersectionOf(A,
complementOf B)`; any other shape fails closed. The scope is a Go list of 17 `catalog/bfo` constants: `BearerOf`,
`InheresIn`, `GenericallyDependsOn`, `IsCarrierOf`, `Realizes`, `RealizedIn`, `Concretizes`, `IsConcretizedBy`,
`HasMemberPart`, `MemberPartOf`, `ContinuantPartOf`, `HasContinuantPart`, `HasOccurrentPart`, `OccurrentPartOf`,
`OccupiesTemporalRegion`, `Precedes`, `PrecededBy`. A scenario test pins three interpreted rows written out
independently (`196` domain IC minus SR; `059` domain Process or SDC; `199` range TemporalRegion). (b) A hand-written
Go table plus a test binding it to the fixture: two copies of the same facts. Choose (a). Membership is decided over
the catalog closure: in set means some member `cco.IsA` the class; "not SR" means no member `cco.IsA` SR (closed
world over the declared closure). A node class that neither catalog knows fails closed.

**D5. Anchoring rule and connectivity.** If the previous class is IC and the node is SDC, the predicate must be
`bfo.BearerOf`; if SDC to IC, `bfo.InheresIn` (Open Question 1 covers CCO sub-properties). Connectivity: (a) opt-in
by pack name, nothing opted in; its negative test is diekm opted in (it names `intel.platform`), and its positive
test is diekm with a seeded `RealizedIn` step to `cco.Act`. (b) Defer to an issue: it has no pack that passes. Choose
(a), as the owner confirmed on 2026-11-07.

**D6. Tests and mutants.** Disjointness sets are read from the fixture (`owl:disjointWith` plus
`AllDisjointClasses`). Bucket roots are catalog constants. `task mutate:check` mutants: (1) the disjointness reader
drops `AllDisjointClasses`; (2) the bucket check reads only `class_iri`; (3) the hop check skips the previous-node
domain on step > 0; (4) the interpreter ignores `complementOf`; (5) the anchoring rule accepts any OBO predicate;
(6) the selector takes every pack; (7) `oracle.Objects` drops blank nodes.

## Risks / Trade-offs

- Closed-world complement: a class whose closure omits SR but is spatial passes. Mitigated by the closure test.
- Fiat-boundary entity types fail the bucket check until an exemption is ruled (Open Question 3).
- `catalog/bfo/bfo.go:142,153` doc comments for `OccupiesTemporalRegion` and `HasMemberPart` disagree with the
  fixture's domains (`bfo-core.ttl:566-570,286-287`); task 4.1 corrects them.

## Migration Plan

Test-only and additive; rollback is reverting the PR.

## Open Questions (owner)

Answered by the owner on 2026-11-07: 1 (a) strict `BearerOf`/`InheresIn`; 2 require a SHACL mapping, failing
closed without one; 3 reject fiat boundaries; 4 leave the other OBO predicates unchecked.

1. D5: is a hop from IC to SDC allowed through a CCO v2.2 sub-property of `BFO_0000196`, such as `ont00001954` "has
   capability" (one exists; three for `084`/`101`)? (a) Strict `BearerOf` only. (b) Accept sub-properties asserted
   in `cco-2.2`. Recommend (a) now, (b) when a pack needs it.
2. D3: must every BFO pack with projections ship a SHACL mapping, or may a test-local subject table stand in?
   Recommend the mapping, failing closed without one.
3. Add the fiat-boundary exemption (`140/142/146/147`) now, or reject them until a pack admits one? Recommend reject.
4. Should OBO predicates outside the 17 (`ParticipatesIn`, `OccursIn`, `LocatedIn`, …) stay unchecked, or join the
   table (the fixture declares their domain and range)? Recommend unchecked (R1 scope).

Architect: claude (claude-opus-5-5, high).
