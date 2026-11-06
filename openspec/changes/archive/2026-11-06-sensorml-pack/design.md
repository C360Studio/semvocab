# Design: SensorML 3.0 and SOSA/SSN Vocabulary Pack

## Context

Issue #26 tracks OGC SWE Common 3.0, OGC SensorML 3.0, and W3C SOSA/SSN support across two phases. Phase 0 and Phase 1
landed on PR #27 (`2026-10-27-swe-sensorml-3-provenance`), pinning:

- W3C SOSA/SSN + `ssn-system` 2017 Recommendation (`w3c/sdw` commit `cb31ee9de636a7101337815ba67ba9af4a7c1348`,
  `ssn/integrated/{sosa,ssn,ssn-system}.ttl`).
- OGC SWE Common 3.0 (`OGC 24-014`, `3.0.0`) and SensorML 3.0 (`OGC 23-000`, `3.0.0`) from
  `opengeospatial/ogcapi-connected-systems` tag `v1.0.0` (`8e03b236a049849f2ccc24b4fd9fdce5ff69bed2`).

Phase 2 scopes `packs/sensorml` for `C360Studio/semconnect` under three binding owner rulings on #26 (`2026-10-27`):

- **R4 (Pack boundary & CS API Part 2 coverage):** Scope `packs/sensorml` strictly to published SensorML 3.0 +
  SOSA/SSN (the 5 OGC CS API Part 1 resources, `sensorml.*` predicates, and Part 1 association predicates), keeping
  Working-Draft CS API Part 2 (`csapi:` streams, commands, events, feasibility, and federation wildcards) in
  `semconnect` or a separate `packs/csapi` follow-up.
- **R5 (Entity-type `<domain>.<type>` keys & class hierarchy alignment):** Retain `csapi.<type>` keys (zero EntityID
  migration for `semconnect`) with dual SensorML 3.0 + SOSA/SSN closures via `local_assertions` in
  `docs/admission-ledger.yaml` (e.g., `sml:PhysicalSystem rdfs:subClassOf ssn:System`).
- **R6 (Deliverable & stop boundary):** Claim a `gemini/` worktree + draft PR (`Addresses #26`), commit the OpenSpec
  proposal/design/tasks + `docs/admission-ledger.yaml` updates, and pause for owner sign-off before implementing
  `packs/sensorml`.

## Goals and Non-Goals

**Goals:**

- Disambiguate `catalog` and `pack` candidates in `internal/ledger` so `catalog: sensorml` and `pack: sensorml` coexist
  cleanly and ambiguous untyped `Entry` lookups fail closed.
- Ship `packs/sensorml` on the exact pattern of `packs/diekm` (`PackSpec` YAML, SHACL shapes within SemVocab's
  documented profile, compiler mapping YAML with projection contracts, side-effect-free `Spec`, and explicit
  `Register`).
- Preserve `semconnect` `setup03a` predicate expectations while fixing `semconnect`'s three IRI defects (`sml:position`,
  `sml:attachedTo`, `ssn-system:hasSystemCapability`) and providing canonical `sensorml.*` predicates for all five OGC
  API — Connected Systems Part 1 resources.
- Verify every pack IRI, superclass closure, SHACL constraint, and projection step against the vendored W3C and OGC
  oracles and SemEngine `v0.1.0-alpha.2`.

**Non-Goals:**

- Admitting any Working-Draft OGC API — Connected Systems Part 2 (`OGC 23-002`) resource, predicate, or `csapi:` IRI.
- Mutating SemEngine's global `vocabulary` or `vocabulary/export` registries outside explicit `Register` calls.
- Exporting duplicate Go IRI constants for standard terms SemEngine (`vocabulary/standards.go`) or `catalog/*` already
  export, or exporting a `Compile` wrapper that would introduce a `packs/sensorml -> shacl` dependency.
- Runtime SPARQL query expansion or RDF export projection (owned by SemSPARQL per issue #1 ruling Q34).

## Decisions

### D1: Ledger Catalog and Pack Disambiguation

`docs/admission-ledger.yaml` already records `- catalog: sensorml`. Adding `- pack: sensorml` makes bare `"sensorml"`
ambiguous:

- `(Entry).Kind() string` returns `"catalog"` or `"pack"` (or `""` if malformed).
- `(Entry).ID() string` returns `"<kind>:<name>"` (e.g., `"catalog:sensorml"`, `"pack:sensorml"`).
- `(Ledger).Catalog(name string) (Entry, error)` and `(Ledger).Pack(name string) (Entry, error)` look up by kind and
  fail closed if zero or multiple entries match.
- `(Ledger).Entry(name string) (Entry, error)` remains for unambiguous names and fails closed with an error naming
  `Catalog` or `Pack` when multiple entries match `name`.
- `internal/ledger/ledger_test.go` splits candidates into 19 `catalogCandidates` and 4 `packCandidates` (`diekm`,
  `bfo-cco-core`, `swe-core`, `sensorml`), and restricts the `typedSourceCatalogs` exemption to `e.Catalog != ""` so
  `- pack: sensorml` must carry `contents`.
- Existing callers in `internal/oracle/schema.go`, `catalog/sensorml/sensorml_test.go`, and `packs/packs_test.go`
  migrate to `Catalog(...)` and `Pack(...)` in Phase 2a so `task verify` is green at the R6 stop boundary.

### D2: Five Disjoint `csapi.<type>` Targets and Cross-Standard `local_assertions`

Per ruling R5, the five entity types keep `csapi.<type>` keys so `semconnect`'s 6-part entity IDs
(`c360.semconnect.systems.csapi.<type>.<instance>`) require zero migration. Each entity type binds a distinct
`class_iri` (`sh:targetClass`), with cross-standard SensorML-to-SOSA/SSN superclasses declared via four
`local_assertions` in `docs/admission-ledger.yaml`:

| Entity Type | `class_iri` (`sh:targetClass`) | Ledger `local_assertions` Axiom | Complete Breadth-First `superclasses` Closure |
| --- | --- | --- | --- |
| `csapi.system` | `sml:PhysicalSystem` | `sml:PhysicalSystem rdfs:subClassOf ssn:System` | `sml:AbstractPhysicalProcess`, `ssn:System`, `sml:AbstractProcess`, `sml:DescribedObject` |
| `csapi.procedure` | `sml:SimpleProcess` | `sml:SimpleProcess rdfs:subClassOf sosa:Procedure` | `sml:AbstractProcess`, `sosa:Procedure`, `sml:DescribedObject` |
| `csapi.deployment` | `sml:Deployment` | `sml:Deployment rdfs:subClassOf ssn:Deployment` | `sml:DescribedObject`, `ssn:Deployment` |
| `csapi.samplingfeature` | `sosa:Sample` | *(none — asserted in `ssn.ttl`)* | `sosa:FeatureOfInterest`, `sosa:Result` |
| `csapi.property` | `sml:DerivedProperty` | `sml:DerivedProperty rdfs:subClassOf sosa:ObservableProperty` | `swe:AbstractSweIdentifiable`, `sosa:ObservableProperty`, `swe:AbstractSWE`, `ssn:Property` |

Alignment hierarchy rationale: in `catalog/sensorml`, `sml:PhysicalSystem` subclasses `sml:AbstractPhysicalProcess`,
which subclasses `sml:AbstractProcess`, while `sml:SimpleProcess` subclasses `sml:AbstractProcess` directly. Aligning
`sml:SimpleProcess` (rather than `sml:AbstractProcess`) to `sosa:Procedure` prevents `sml:PhysicalSystem` from falsely
inheriting `sosa:Procedure`, preserving the SSN distinction where an `ssn:System` implements a `sosa:Procedure`.
Aligning `sml:DerivedProperty` to `sosa:ObservableProperty` transitively yields `ssn:Property` from `ssn.ttl`.

### D3: Collision-Free Predicate IRIs, Projections, and Consumer Migration

`shacl/compile.go` (`indexMapping`) requires a 1-to-1 bijection between SHACL `sh:path` IRIs and mapped predicate names.
`packs/sensorml` declares 20 `sensorml.<category>.<property>` predicates over 20 distinct standard and Phase 1 catalog
IRIs (`""` denotes untyped polymorphic values per SemEngine `vocabulary/predicates.go:208`):

| # | Dotted Predicate | `iri` | `data_type` | `inverse_of` / `projection` |
| --- | --- | --- | --- | --- |
| 1 | `sensorml.process.type` | `rdf:type` | `string` | — |
| 2 | `sensorml.process.uid` | `dc:identifier` | `string` | — (`vocabulary.DcIdentifier`) |
| 3 | `sensorml.process.label` | `dc:title` | `string` | — (`vocabulary.DcTitle`) |
| 4 | `sensorml.process.description` | `dc:description` | `string` | — |
| 5 | `sensorml.process.definition` | `rdfs:isDefinedBy` | `string` | — |
| 6 | `sensorml.system.hosts` | `sosa:hosts` | `entity_id` | `inverse_of: sensorml.component.is-hosted-by` |
| 7 | `sensorml.component.is-hosted-by` | `sosa:isHostedBy` | `entity_id` | `inverse_of: sensorml.system.hosts` |
| 8 | `sensorml.process.has-sub-system` | `ssn:hasSubSystem` | `entity_id` | — |
| 9 | `sensorml.process.used-procedure` | `sosa:usedProcedure` | `string` | — |
| 10 | `sensorml.process.attached-to` | `sml:attachedTo` | `string` | — (fixes `sosa:isHostedBy` collision) |
| 11 | `sensorml.process.position` | `sml:position` | `json` | — (fixes non-standard `sosa:hasLocation`) |
| 12 | `sensorml.identifier.value` | `skos:notation` | `""` | — (`vocabulary.SkosNotation`) |
| 13 | `sensorml.capability.value` | `ssn-system:hasSystemCapability` | `""` | 1-step: `ssn-system:hasSystemCapability -> sml:CapabilityList -> swe:value` |
| 14 | `sensorml.characteristic.value` | `ssn:hasProperty` | `""` | 1-step: `ssn:hasProperty -> sml:CharacteristicList -> swe:value` |
| 15 | `sensorml.deployment.deployed-system` | `ssn:deployedSystem` | `string` | — (replaces `cs-api.deployment.deployed-systems`) |
| 16 | `sensorml.deployment.parent` | `skos:broader` | `entity_id` | — (`vocabulary.SkosBroader`; replaces `cs-api.deployment.parent`) |
| 17 | `sensorml.sampling.sampled-feature` | `sosa:isSampleOf` | `string` | — (`samplingFeature.json` `sampledFeature@link`) |
| 18 | `sensorml.sampling.hosted-procedure` | `ssn:implements` | `string` | — (replaces `cs-api.samplingfeature.hosted-procedure` without `sosa:usedProcedure` collision) |
| 19 | `sensorml.property.definition` | `sml:definition` | `string` | — (replaces `cs-api.property.definition` without `rdfs:isDefinedBy` collision) |
| 20 | `sensorml.property.base-property` | `sml:baseProperty` | `string` | — (replaces `cs-api.property.base-property`) |

Predicates 1, 2, 3, and 6 preserve `semconnect`'s `setup03a` reference expectations byte-for-byte. Predicates 10, 11,
and 13 eliminate `semconnect`'s three IRI defects:

1. `sosa:hasLocation` -> `sml:position` (`data_type: json`), replacing an unasserted SOSA IRI with SensorML 3.0's
   declared position property (`sensorml.PropPosition`).
2. `sosa:isHostedBy` on `sensorml.process.attached-to` -> `sml:attachedTo` (`sensorml.PropAttachedTo`), separating
   physical platform attachment from logical `sosa:hosts` / `sosa:isHostedBy` component hosting.
3. `ssn:hasProperty` on `sensorml.capability.value` -> `ssn-system:hasSystemCapability` (`ssn.HasSystemCapability`),
   separating system capability lists (`sml:CapabilityList -> swe:value`) from characteristic lists
   (`ssn:hasProperty -> sml:CharacteristicList -> swe:value`).

### D4: Prefix Compatibility with SemEngine `defaultPrefixes`

SemEngine's `vocabulary/export/prefix.go:29` pre-registers `"dc": "http://purl.org/dc/terms/"`. Binding `"dcterms"` to
the same namespace would map two prefixes to one IRI stem. `sensorml.yaml` and `sensorml.shacl.ttl` declare `"dc"`
alongside `"rdf"`, `"rdfs"`, `"skos"`, `"sosa"`, `"ssn"`, `"ssn-system"`, `"swe"`, and `"sml"`. Registering
`packs/swe-core` and `packs/sensorml` in the same process is idempotent and conflict-free.

### D5: Package Surface, Open SHACL Shapes, and Reconcile Contracts

`packs/sensorml` mirrors `packs/diekm` (`packs/diekm/diekm.go`):

- **Go API:** exports `Spec() (vocabulary.PackSpec, error)`, `Register(opts ...vocabulary.PackOption) error`,
  `Shapes() []byte`, and `Mapping() []byte`. Omitting a `shacl` import in `packs/sensorml` allows `shacl` test suites
  to import `packs/sensorml` without circular dependencies.
- **Open SHACL shapes:** `sensorml.shacl.ttl` defines `sml:PhysicalSystemShape`, `sml:SimpleProcessShape`,
  `sml:DeploymentShape`, `sml:SamplingFeatureShape`, and `sml:DerivedPropertyShape` with `sh:closed` omitted so
  SemEngine framework predicates (`entity.indexing.profile`, `geo.location.*`) never trigger closed-shape violations.
- **IRI-valued `string` paths:** `shacl/terms.go:246-256` (`classifyString`) classifies absolute IRI strings and 6-part
  EntityIDs as `termIRI` rather than `xsd:string` literals. IRI-bearing `string` paths (`rdf:type`, `dc:identifier`,
  `rdfs:isDefinedBy`, `sosa:usedProcedure`, `sml:attachedTo`, `ssn:deployedSystem`, `sosa:isSampleOf`, `ssn:implements`,
  `sml:definition`, `sml:baseProperty`) therefore omit `sh:datatype` in `sensorml.shacl.ttl` (retaining `data_type:
  string` from `sensorml.yaml` via `fillDatatypes`) and constrain cardinality plus XSD-subset `sh:pattern`.
- **`entity_id` paths:** `sosa:hosts`, `sosa:isHostedBy`, `ssn:hasSubSystem`, and `skos:broader` carry cardinality
  constraints only (`shacl/compile.go:480-483` forbids `sh:datatype` on `entity_id` predicates).
- **Position constraints (`sml:position`):** carries `sh:datatype rdf:JSON ; sh:maxCount 1` on `PhysicalSystemShape`,
  `DeploymentShape`, and `SamplingFeatureShape` (`shacl/terms.go:217-220` classifies valid JSON strings as `rdf:JSON`
  and malformed JSON as `xsd:string`, rejecting invalid JSON), and **`sh:maxCount 0`** on `SimpleProcessShape` to
  enforce OGC `/req/procedure/location`.
- **Reconcile projection contracts:** `sensorml.mapping.yaml` binds all five `csapi.*` entity types with
  `birth: [sensorml.process.type]` and a single `reconcile` group `representation` containing every remaining predicate
  its shape constrains (including `sensorml.process.position` on `csapi.procedure`, satisfying ruling Q19 in
  `shacl/compile.go:716-733` and leaving `Bundle.Unwritable()` empty).

## Risks and Trade-offs

- **`SamplingFeature` optional `dc:title` vs `DescribedObject` required `dc:title`:** OGC SensorML 3.0
  `DescribedObject.json` (`PhysicalSystem`, `SimpleProcess`, `Deployment`) and `DerivedProperty.json` require `label`
  (`dc:title`, `sh:minCount 1 ; sh:maxCount 1`), whereas OGC CS API Part 1 GeoJSON `samplingFeature.json` requires
  `uid` (`dc:identifier`) and `sampledFeature@link` (`sosa:isSampleOf`) with optional `name`/`label`.
  `SamplingFeatureShape` keeps `dc:title` at `sh:maxCount 1` (`sh:minCount 0`) so minimal conformant Part 1 payloads
  validate cleanly.
- **Phase 2a caller disambiguation:** Updating `internal/ledger` and its existing test/oracle callers in Phase 2a ahead
  of `packs/sensorml` code ensures `task verify` passes at the Ruling R6 sign-off boundary.
