# Design: phase-0-provenance

## Context

Inputs, each read at a named revision:

- SemEngine `e60229334b8d8da3c0350e17802e2ea9631f91c8` (`v0.1.0-alpha.1`)
- SemStreams `8b99efe9c66a4faa4fa509f9f62cc6bad8392128`
- SemSPARQL `origin/main` `b64f35e`: no Go code yet
- BFO-2020 `release-2024-01-29`
- CCO `v1.4-2023-04-07`, `v1.5-2024-02-14`, `v1.7-2024-11-03`, `v2.0-2024-11-06`

"Closure" below means the transitive closure of asserted named `rdfs:subClassOf`, without `owl:Thing`.
`owl:equivalentClass` and restriction axioms are not evaluated.

## Goals and non-goals

Goal: every admitted term traces to a source at an exact revision, and every later phase has an observable acceptance
test and a named dependency gate. Non-goals: implementation, and any Phase 4 rule work.

## Decisions

### D1. SemEngine pin

`v0.1.0-alpha.1` = `e60229334b8d8da3c0350e17802e2ea9631f91c8`. It provides `LoadPackYAML`/`LoadPackJSON`
(`vocabulary/pack.go:135-177`), `RegisterPack` (`:183`), `WithPrefixRegistrar` (`:127`), and
`export.RegisterPrefixes` (`vocabulary/export/prefix.go:63`). It lacks Stage 1.1. Phase 1 adds the `go.mod`
require; Phase 0 cannot, because `go mod tidy` would drop an unused require and `task tidy:check` would fail.

### D2. BFO release

BFO 2020 core, BFO-2020 tag `release-2024-01-29` (`044490fc`), `src/owl/bfo-core.ttl`, version IRI
`http://purl.obolibrary.org/obo/bfo/2020/bfo-core.ttl`, CC-BY-4.0. SemStreams' hierarchy already encodes BFO 2020
terms (`History` = `BFO_0000182`), so this release fits the carried material best. SemSPARQL#1's summary says
"BFO 2.0"; that wording should become BFO 2020.

### D3. CCO release (owner ruling Q1)

Proposed: `v1.7-2024-11-03`, the last release in the `http://www.ontologyrepository.com/CommonCoreOntologies/`
namespace used by SemStreams and #1, built on the D2 BFO core. CCO 1.4, which SemStreams cites, embeds BFO
`2019-08-26` plus RO and conflicts with D2. CCO 2.x uses opaque `https://www.commoncoreontologies.org/ont…` IRIs,
so adopting it is a namespace change that needs a mapping. For every term this change examined, `v1.5` and `v1.7`
gave the same results.

### D4. Ports admit only release-verified terms; hierarchy follows the release

The SemStreams `bfo` and `cco` packages carry IRIs absent from, or mislabelled in, the releases they cite (ledger
`excluded_source_symbols`). A port carries the verified symbols with SemStreams' names and IRIs. Parent links come
from the release, and every divergence from the SemStreams map is listed in the ledger. Minting non-CCO terms in the
CCO namespace is rejected.

### D5. Full ancestor closure in `Superclasses`

SemSPARQL#1 §2.2.3 requires every pack to register the full ancestor closure, because `LookupEntityTypesByClassIRI`
(`vocabulary/entity_type.go:186-205`) matches only `ClassIRI` and the directly listed `Superclasses`. SemSPARQL's
startup check only fires when a superclass is itself a registered `ClassIRI`, so it does not catch a short closure
against unregistered BFO/CCO classes. SemVocab therefore proves the closure against the ontology release. Order is
nearest first, then lexical among equals.

Closures in CCO `v1.7-2024-11-03` (v1.5 gives the same):

- `cco:Vehicle`: `cco:Artifact`, `BFO_0000040`, `BFO_0000004`, `BFO_0000002`, `BFO_0000001`. CCO does not entail
  `BFO_0000030` (object) for Vehicle; #1's fixture asserts it (Q2).
- `cco:ActOfObservation`: `cco:IntentionalAct`, `cco:Act`, `BFO_0000015`, `BFO_0000003`, `BFO_0000001`.
- `cco:Altitude`: `cco:Height`, `cco:OneDimensionalExtent`, `cco:SizeQuality`, `BFO_0000019`, `BFO_0000020`,
  `BFO_0000002`, `BFO_0000001`.
- `cco:MeasurementInformationContentEntity`: `cco:DescriptiveInformationContentEntity`,
  `cco:InformationContentEntity`, `BFO_0000031`, `BFO_0000002`, `BFO_0000001`.

### D6. Superclass IRIs are expanded `http(s)` IRIs

SemEngine accepts any string with a syntactically valid scheme (`vocabulary/projection.go:91-119`, called from
`pack.go:336-343`), so `cco:Vehicle` would register as an "absolute IRI". SemVocab's pack test asserts every
`class_iri`, `superclasses` entry, `iri`, and projection IRI begins with `http://` or `https://` and lies in a
declared prefix namespace. This checks pack content and does not copy SemEngine's validation.

### D7. DIEKM namespace (owner ruling Q5)

`https://w3id.org/c360/diekm#` is a C360-local namespace. On 2026-10-15, `https://w3id.org/c360/diekm` returned
404 and `perma-id/w3id.org` had no `c360/` directory. Its IRIs do not dereference. Documentation calls it a C360
local vocabulary and never an externally registered or certified ontology. Registering it would be an owner action:
a PR to `perma-id/w3id.org`.

### D8. Re-registration semantics are SemEngine's

At the pin, `RegisterPack` overlays scalar fields last-writer-wins (`pack.go:271-309`, `:330-335`), unions
`Superclasses` (`:340`), and errors only on validation or inverse mismatch (`registry.go:379-414`). Phase 1 tests
record this behaviour; they do not change it.

## Corrections to the issue #1 DIEKM fixture

Checked against CCO 1.4, 1.5 and 1.7 and BFO 2020 core:

- `cco:has_role` (predicate `iri`) and `cco:AgentRole` (`node_class_iri`) are not declared in any of them (Q3).
- `RO_0000087` and `RO_0000053` are RO terms, absent from BFO 2020 core and CCO 1.5+; BFO 2020's counterpart of
  `RO_0000053` is `BFO_0000196` ("bearer of") (Q3).
- `cco:has_decimal_value` has `rdfs:domain cco:InformationBearingEntity` in CCO 1.7, not the MICE node the fixture
  attaches it to (Q4).
- Neither entity type lists its full ancestor closure (D5).

## Adopter seam inventory

| Surface | Caller | Commit | Passes | Relies on |
| --- | --- | --- | --- | --- |
| `packs/diekm/diekm.yaml` (embedded) | SemEngine `LoadPackYAML`, `RegisterPack` | `e602293` | YAML bytes; `WithPrefixRegistrar(export.RegisterPrefixes)` | known fields only (`pack.go:162-163`); atomic failure |
| DIEKM registration function (Phase 1 names it; proposed `diekm.Register(opts ...vocabulary.PackOption) error` plus `diekm.Spec() (vocabulary.PackSpec, error)`) | none yet; SemSPARQL#1 tasks 3.1 and 4.2 | SemSPARQL `b64f35e` has no Go code | composition root call | full ancestor closure; DIEKM projection specs |
| `catalog/bfo`, `catalog/cco` `Parents`/`IsA` | none in-repo; semsource (a #1 consumer) | none observed | class IRIs | release-correct closure |
| SemVocab `v0.1.0` tag | SemSPARQL#1 task 0.1 | — | `go.mod` require | tag after Phase 1 (owner ruling) |

## Dependency order and phase plan

Order: Phase 0 (this change) → Phase 1 → tag `v0.1.0` → Phase 2 → Phase 3. Phase 4 is deferred. Each later phase
opens its own OpenSpec change. Its `Hold:` lines name the gates below, so this change's tasks can all be ticked
before archive. Overlapping open work: semengine PR #261 (draft; Stage 1.1, gates Phase 2); semengine#259 (gates
SemSPARQL task 4.1, which Phase 3's end-to-end route uses). No open SemVocab PR overlaps.

### Phase 1: catalogs and built-in packs

Gate: available (`v0.1.0-alpha.1`). Scope: `catalog/bfo`, `catalog/cco` (ledger lists), `packs/bfo-cco-core.yaml`,
`packs/diekm/diekm.yaml`, initial `packs/diekm/diekm.shacl.ttl`, and the `go.mod` require.

Acceptance tests:

- **`TestCatalogPortParity`:**
  - Each admitted constant equals its SemStreams-pin IRI, using a fixture extracted from the pin by a recorded
    command.
  - Each admitted term is declared in the vendored release file, matched by sha256, with the expected `rdf:type`.
  - Each excluded source symbol is absent from the catalog.
  - It fails when one IRI is altered (paired sensitivity test).
- **`TestCatalogHierarchyMatchesRelease`:** for every admitted class, `Parents` equals the release closure. The
  ledger's SemStreams divergences are asserted as such.
- **`TestPackSuperclassClosure`:**
  - For every entity type in every embedded pack, the `superclasses` set equals the release closure of its declared
    parents, plus any recorded DIEKM-local assertion and that assertion's closure.
  - Every IRI is an expanded `http(s)` IRI in a declared namespace (D6).
  - A pack missing one ancestor fails.
- **`TestDIEKMPackLoadsAndRegisters`:**
  - The exact embedded YAML goes through `LoadPackYAML`, then
    `RegisterPack(..., WithPrefixRegistrar(export.RegisterPrefixes))`.
  - `GetEntityTypeMetadata`, `GetPredicateMetadata` (including `Projection`), and
    `LookupEntityTypesByClassIRI(<each closure IRI>)` return the declared values.
  - The `diekm` prefix compacts in Turtle output.
- **JSON/YAML equivalence:** both forms decode to equal `PackSpec` values.
- **Atomic failure:** an unknown field fails to load, and an invalid entry fails `RegisterPack`. After a failure,
  registry getters and serializer prefix output are unchanged; the tests assert state and never restore it.
- **Repeated and conflicting registration:** a repeated identical registration leaves state equal. A conflicting
  `ClassIRI` follows D8, documented and tested.
- **Direct serialization:** Turtle, N-Triples, and JSON-LD of direct triples parse. These tests are not
  projection-expansion or SHACL tests.
- `task verify` passes.

### Phase 2: bounded SHACL compilation

`Hold:` semengine#256 merged and released in a tag that SemVocab pins. Acceptance: #1 Phase 2 list, unchanged.
Highlights: `TestSHACLCompilerOutputsValidPackAndContracts` runs through the Stage 1.1 pure-validation API with
global state unchanged; the entity pattern accepts `acme.edge.sensor.intel.platform.p1` and rejects
`acme.edge.intel.other.platform.p1`; maxCount, enumeration, reconcile/append, and birth cases; Rapid determinism
and fuzz with reproducible seeds; measured benchmarks; `docs/shacl-compilation.md` and the support matrix.

### Phase 3: export projection and external SHACL validation

Owning work: SemSPARQL#1 task 3.1 (virtual `rdf:type` and the closure startup check), task 3.3 (CONSTRUCT/DESCRIBE
unfolding, `POST /export`, `docs/shacl-compliance-gate.md`; needs semengine#256), and task 4.2 (end-to-end with
SemVocab's `diekm` pack).

Callable API as specified, none of it existing at `b64f35e`:

- `UnfoldEntity(entity)` in `sparql/construct.go`
- `POST /export` with `Accept: text/turtle | application/n-triples | application/ld+json` in `server/handler.go`
- serialization via `export.Serialize`

`Hold:` SemSPARQL#1 tasks 3.3 and 4.2 merged and tagged, and semengine#256 merged. Before integration, record the
merged SemSPARQL revision and the exact callable signature.

Acceptance: #1 Phase 3 list, including:

- a nonzero focus-node set asserted independently of `sh:conforms`
- valid and deliberately invalid fixtures
- native and external validator agreement on the shared profile
- typed-literal regressions (#252 fix present in `e602293`)
- a pinned external validator with a recorded version and options

### Phase 4: rule translation (deferred)

Deferred until `processor/rule` and `pkg/rulepack` exist in SemEngine. The Phase 2 compiler rejects `sh:rule`.

## Declared costs

- The ontology oracles are vendored release files: CCO merged TTL about 1.45 MB, BFO core about 100 KB. They carry
  the CC-BY-4.0 attribution and the BSD-3-Clause notice. The test-only Turtle reader is a Phase 1 design choice (Q8).
- The SemStreams parity fixture is generated from the pin by a recorded command, not by importing the SemStreams
  module.

## Open questions (owner, on #1)

- **Q1 CCO release:** `v1.7-2024-11-03` (proposed) or `v1.5-2024-02-14`. CCO 2.x is rejected for its namespace change.
- **Q2 `intel.platform`:** keep `BFO_0000030` as a recorded DIEKM-local assertion (semsparql#1 tests depend on it),
  or drop it.
- **Q3 role predicate:** replace `cco:has_role`, `cco:AgentRole` and `RO_0000087` with a DIEKM-local predicate
  projected via `BFO_0000196` → `BFO_0000023` (or a real CCO role class such as `cco:OperatorRole`), or adopt and pin
  an RO release as a third ontology. The same choice decides whether `RO_0000053` becomes `BFO_0000196`.
- **Q4 altitude terminal:** keep `has_decimal_value` on the MICE node, or add a hop to an InformationBearingEntity.
- **Q5 DIEKM namespace:** register `w3id.org/c360`, or keep it local and unregistered. Either way, no certification
  claim.
- **Q6 `bfo-cco-core` pack contents:** prefixes only (proposed), or more.
- **Q7 ledger shape test:** add `TestAdmissionLedgerShape` now, or in Phase 1 (proposed).
- **Q8 test oracle form:** vendor the release TTLs plus a test-only Turtle reader (proposed), or a generated closure
  fixture.
- **Q9 prefix naming:** prefix `bfo` for `http://purl.obolibrary.org/obo/` compacts RO terms as `bfo:RO_…`; `obo`
  would be more accurate.
