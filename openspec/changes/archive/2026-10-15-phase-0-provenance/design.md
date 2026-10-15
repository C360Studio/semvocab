# Design: phase-0-provenance

## Context

Inputs, each read at a named revision:

- SemEngine `e60229334b8d8da3c0350e17802e2ea9631f91c8` (`v0.1.0-alpha.1`)
- SemStreams `8b99efe9c66a4faa4fa509f9f62cc6bad8392128`
- SemSPARQL `origin/main` `b64f35e`: no Go code yet
- BFO-2020 `release-2024-01-29`
- CCO `v1.4-2023-04-07`, `v1.5-2024-02-14`, `v1.7-2024-11-03`, `v2.0-2024-11-06`, `v2.2` (pinned)

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

### D3. CCO release

Owner ruling Q1 (2026-10-15): CCO v2. Pinned to `v2.2` (commit `0bc7d33e`, merged TTL sha256 `a9453382…`), the
latest release; no evidence was found that DIEKM targets another 2.x. v2 uses opaque IRIs under
`https://www.commoncoreontologies.org/`, so no SemStreams-pin CCO IRI is admitted. The `cco` catalog is new content
from v2.2; SemStreams' Go names are kept, except the six terms whose v2.2 `rdfs:label` differs from SemStreams'
name, which take the v2.2 label (Q10). The ledger records each term's v2.2 IRI and `rdfs:label`, the table the Phase
1 oracle checks. CCO publishes no 1.x→2.x crosswalk.
The mapping was derived by identical `rdfs:label` plus definition between v1.7 and v2.0, then by IRI identity from
v2.0 to v2.2, and Phase 1 tests it against the v2.2 file. Deprecated v2.2 terms are not admitted. The BFO pin is
BFO-2020 commit `5ed46876`, byte-identical to the BFO core v2.2 imports and axiom-identical to `release-2024-01-29`.

### D4. Ports admit only release-verified terms; hierarchy follows the release

The SemStreams `bfo` and `cco` packages carry IRIs absent from, or mislabelled in, the releases they cite (ledger
`excluded_source_symbols`). The `bfo` port carries the verified symbols with SemStreams' names and IRIs; `cco` keeps
only SemStreams' names (D3). Parent links come
from the release, and every divergence from the SemStreams map is listed in the ledger. Minting non-CCO terms in the
CCO namespace is rejected.

### D5. Full ancestor closure in `Superclasses`

SemSPARQL#1 §2.2.3 requires each pack to register its full ancestor closure, because `LookupEntityTypesByClassIRI`
(`vocabulary/entity_type.go:186-205`) matches only `ClassIRI` and the directly listed `Superclasses`. SemSPARQL's
startup check fires only when a superclass is itself a registered `ClassIRI`, so it does not catch a short closure
over unregistered CCO/BFO classes. SemVocab therefore proves closure against the CCO v2.2 merged file, which embeds
BFO. Order is declared parents first, then breadth-first. A DIEKM-local assertion (`IntelligencePlatform ⊑
BFO_0000030`, ruling Q2) is listed in the ledger and its own closure is included.

Closures in CCO v2.2 (`cco:` = `https://www.commoncoreontologies.org/`, `obo:` = `http://purl.obolibrary.org/obo/`):

- Vehicle `cco:ont00000713`: `cco:ont00000995` (Material Artifact), `BFO_0000040`, `BFO_0000004`, `BFO_0000002`,
  `BFO_0000001`. `BFO_0000030` is not an ancestor; DIEKM asserts it locally, so `intel.platform` lists 7 IRIs.
- Act of Observation `cco:ont00000037`: `cco:ont00000228` (Planned Act), `cco:ont00000005` (Act), `BFO_0000015`,
  `BFO_0000003`, `BFO_0000001`; `intel.observation` lists 6 IRIs.
- Altitude `cco:ont00000787`: `cco:ont00000967` (Height), `cco:ont00001367` (One Dimensional Extent),
  `cco:ont00001202` (Size Quality), `BFO_0000019`, `BFO_0000020`, `BFO_0000002`, `BFO_0000001`.
- Measurement Information Content Entity `cco:ont00001163`: `cco:ont00000853`, `cco:ont00000958`, `BFO_0000031`,
  `BFO_0000002`, `BFO_0000001`.
- Information Bearing Entity `cco:ont00000253`: `BFO_0000030`, `BFO_0000040`, `BFO_0000004`, `BFO_0000002`,
  `BFO_0000001`.

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

Checked against CCO 1.4, 1.5, 1.7 and 2.2 and BFO 2020 core; rulings Q1–Q5 of 2026-10-15 applied:

- CCO namespace `http://www.ontologyrepository.com/CommonCoreOntologies/` (CCO ≤1.7) becomes
  `https://www.commoncoreontologies.org/` (CCO v2.2, Q1). Every `cco:` term becomes an opaque IRI: Vehicle →
  `ont00000713`, ActOfObservation → `ont00000037`, Altitude → `ont00000787`, is_measured_by → `ont00001904`,
  MeasurementInformationContentEntity → `ont00001163`.
- `cco:has_role`, `cco:AgentRole` and `RO_0000087` exist in no CCO release checked. They are replaced by
  `diekm:assignedRole`, projected via `BFO_0000196` (bearer of) → `BFO_0000023` (role) (Q3).
- `RO_0000053` becomes `BFO_0000196` in the altitude chain (Q3).
- `has_decimal_value` belongs on an Information Bearing Entity, not on the MICE. The chain gains the hop MICE
  `BFO_0000084` (generically depends on) → `cco:ont00000253` (Q4). `BFO_0000058` (is concretized by) would be wrong:
  it targets a specifically dependent continuant. `has decimal value` (`ont00001769`) is deprecated in v2.2 without
  replacement, so the terminal value predicate is the DIEKM-local datatype property `diekm:altitudeMetersValue`
  (domain `cco:ont00000253`, range `xsd:decimal`), as CCO's change note advises (Q4b).
- `BFO_0000030` is not a CCO ancestor of Vehicle; it stays as a recorded DIEKM-local assertion (Q2).
- Superclasses list the full closure (D5).
- The DIEKM namespace stays local and unregistered (Q5).

SemSPARQL#1 text that the namespace change affects (sister repository; reported to its owner, not edited here): the
summary's "BFO 2.0", the §2.3 folding example (opaque `cco:` IRIs, `BFO_0000196`, and the third hop, which
`TestBGPFoldingCollapsesDIEKMChains` must cover), and §3's `?s a cco:Vehicle` (now `cco:ont00000713`).

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
  - Each admitted `bfo` constant equals its SemStreams-pin IRI, using a fixture extracted from the pin by a recorded
    command. `cco` is new content: each constant's `rdfs:label` in the v2.2 file equals the ledger's recorded label.
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

- The ontology oracles are vendored release files: CCO v2.2 merged TTL about 2.0 MB, BFO core about 100 KB. They carry
  the CC-BY-4.0 attribution and the BSD-3-Clause notice. The test-only Turtle reader is a Phase 1 design choice (Q8).
- The SemStreams parity fixture is generated from the pin by a recorded command, not by importing the SemStreams
  module.

## Owner rulings and open questions

Ruled on #1 (2026-10-15): Q1 CCO v2 (pinned v2.2); Q2 keep `BFO_0000030` as a DIEKM-local assertion; Q3
`diekm:assignedRole` via `BFO_0000196` → `BFO_0000023`; Q4 add the Information Bearing Entity hop; Q5 the DIEKM
namespace stays local and unregistered; Q4b the altitude terminal is `diekm:altitudeMetersValue`; Q10 CCO terms whose
v2.2 label differs from SemStreams' name take the v2.2 label as Go name (`PrescriptiveInformationContentEntity`,
`MaterialCopyOfDocument`, `ActOfMaterialArtifactProcessing`, and, applying the same rule after review,
`Plan`, `PlannedAct`, `MaterialArtifact`).

Phase 1 design choices, with proposed defaults; Phase 1's change records the choice:

- **Q6 `bfo-cco-core` pack contents:** prefixes only (proposed), or more.
- **Q7 ledger shape test:** add `TestAdmissionLedgerShape` now, or in Phase 1 (proposed).
- **Q8 test oracle form:** vendor the release TTLs plus a test-only Turtle reader (proposed), or a generated closure
  fixture.
- **Q9 prefix naming:** prefix `bfo` for `http://purl.obolibrary.org/obo/` would compact other OBO terms as `bfo:…`;
  `obo` would be more accurate.
