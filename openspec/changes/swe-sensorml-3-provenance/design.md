# Design: swe-sensorml-3-provenance

## Context and Inventory

Owner rulings R1–R3 on #26 (2026-10-27) fix the standards and delivery boundaries:

- **R1 (Standards):** OGC SWE Common 3.0 (`OGC 24-014`, `3.0.0`) and SensorML 3.0 (`OGC 23-000`, `3.0.0`) from
  `opengeospatial/ogcapi-connected-systems` tag `v1.0.0` (`8e03b236a049849f2ccc24b4fd9fdce5ff69bed2`), paired with W3C
  SOSA/SSN + `ssn-system` 2017 Recommendation (`w3c/sdw` commit `cb31ee9de636a7101337815ba67ba9af4a7c1348`,
  `ssn/integrated/{sosa,ssn,ssn-system}.ttl`).
- **R2 (Phasing):** Phase 0 lands provenance and ledger tests and stops at the draft PR for owner sign-off; Phase 1
  delivers `catalog/{sosa,ssn,swe,sensorml}`, `packs/swe-core`, and vendored oracles; Phase 2 defers `packs/sensorml`
  predicates, projections, and SHACL shapes until consumer entity types and fields settle.
- **R3 (IRI stems):** Mint SWE 3.0 and SensorML 3.0 RDF IRIs under `http://www.opengis.net/spec/SWE/3.0/` and
  `http://www.opengis.net/spec/sensorML/3.0/`, declared in the ledger as a C360 RDF convention over OGC's Clause 5
  normative specification base URIs (`http://www.opengis.net/spec/SWE/3.0` and
  `http://www.opengis.net/spec/sensorML/3.0`), never presented as an externally certified OGC OWL ontology.

Inventory read:

- `C360Studio/semvocab` at `c8a07be9003100a5b498b25cf057edd659b47bb7`; open PR #25 (`package.json` / `package-lock.json`
  only, zero file overlap).
- `C360Studio/semengine` at `v0.1.0-alpha.2` (`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`): zero exported SOSA, SSN,
  `ssn-system`, SWE, or SensorML constants; built-in serializer prefixes `ssn` (`vocabulary/export/prefix.go:33`) and
  `sosa` (`:34`); prose mention at `vocabulary/doc.go:31`; zero `ssn-system`, `swe`, or `sml` prefixes.
- `C360Studio/semstreams` at `8b99efe9c66a4faa4fa509f9f62cc6bad8392128`: `vocabulary/README.md:20,364-366` doc examples
  only; zero SOSA/SSN/SWE/SensorML Go symbols.
- `C360Studio/semsparql` at `e7f827c445e0e11c990869f5ef70550a91880482` and `C360Studio/semconnect` at
  `d0d06e00bf05a545f30ceea798db1c2b1ee47d4f` (`vocabulary/sosa/iris.go`, `vocabulary/swe/iris.go`, read-only consumer
  comparison).

## Adopter Seams

| Seam | Target Package / Artifact | Consumer | Contract |
| --- | --- | --- | --- |
| SOSA 2017 constants | `catalog/sosa` | `catalog/ssn`; `semconnect`; SWE/SOSA consumers | `Namespace` + 13 classes + 21 object properties + 2 datatype properties; constants only (`sosa.ttl` has zero `rdfs:subClassOf` triples) |
| SSN + `ssn-system` 2017 | `catalog/ssn` | SWE/SensorML/SSN consumers; `semconnect` | `Namespace`, `SystemsNamespace`, 29 classes, 23 object properties; `DirectParents`, `Parents`, `IsA` over integrated `sosa` + `ssn` + `ssn-system` |
| SWE Common 3.0 | `catalog/swe` | `catalog/sensorml`; SWE Core consumers; `semconnect` 3.0 migration | `Namespace`, 35 object classes (`PascalCase`), 55 JSON properties (`Prop<PascalCase>`), `SubClassOf`, `Parents`, `IsA` from JSON Schema `allOf` |
| SensorML 3.0 | `catalog/sensorml` | Connected Systems consumers; Phase 2 `packs/sensorml` | `Namespace`, 35 object classes (`PascalCase`), 94 JSON properties (`Prop<PascalCase>`), `SubClassOf`, `Parents`, `IsA` crossing into `catalog/swe` |
| `swe-core` prefix pack | `packs/swe-core.yaml`, `packs.SWECoreSpec`, `packs.RegisterSWECore` | SemEngine + SemSPARQL composition roots | Registers `sosa`, `ssn`, `ssn-system`, `swe`, `sml` via `vocabulary.RegisterPack` + `WithPrefixRegistrar`; atomic rollback on error |
| `packs/sensorml` | Deferred to Phase 2 (Ruling R2) | Connected Systems consumers | Deferred until consumer entity types and fields settle |

## Decisions

### D1: Schema-bundle pins and `created` dates on unversioned W3C TTLs

SWE Common 3.0 and SensorML 3.0 are Draft 2020-12 JSON Schema encoding bundles, not OWL ontologies. Recording them
under `pins.schemas` (`SchemaBundle`) rather than `pins.ontologies` preserves the distinction in structure as well as
prose. `swe-3.0` pins all 23 `swecommon/schemas/json/*.json` files plus `LICENSE`; `sensorml-3.0` pins all 20
`sensorml/schemas/json/*.json` files plus `common/{link,timeInstant,timeInstantOrNow,timePeriod}.json` (referenced via
`../../../common/` from `sensorml/schemas/json/commonDefs.json`) plus `LICENSE`.

W3C's 2017 `sosa.ttl`, `ssn.ttl`, and `ssn-system.ttl` omit `owl:versionIRI` and assert `dcterms:created` (`2017-04-17`
and `2017-05-14`). `ledger.Ontology` adds `created` (`yaml:"created"`), and `checkShape` requires every ontology pin to
carry `version_iri` or `created` rather than inventing a non-existent `version_iri`.

### D2: Collision-free Go identifier mapping

- `catalog/sosa` and `catalog/ssn`: classes and properties use standard `PascalCase` Go identifiers (`sosa.Sensor`,
  `sosa.MadeBySensor`, `ssn.System`, `ssn.Accuracy`, `ssn.HasSystemCapability`). All 37 `sosa` identifiers and 54
  `ssn` + `ssn-system` identifiers are disjoint.
- `catalog/swe` and `catalog/sensorml`: JSON Schema object types (`PascalCase`) and JSON property keys (`camelCase`)
  collide under naive `PascalCase` (`XMLEncoding`'s `"namespace"` vs `swe.Namespace`; `Document`'s `"link"` vs
  `sensorml.Link`). Both JSON Schema catalogs therefore export object classes as `PascalCase` (`swe.Quantity`,
  `sensorml.PhysicalSystem`, `sensorml.Link`) and JSON properties as `Prop<PascalCase>` (`swe.PropUOM`,
  `swe.PropNamespace`, `sensorml.PropLink`), with IRIs preserving the exact JSON Schema local names.
- Upstream spelling fidelity: `swe.AbstractSweIdentifiable` follows the canonical file
  `swecommon/schemas/json/AbstractSweIdentifiable.json` (`sweCommonDefs.json`'s `AbstractSWEIdentifiable` `$ref` alias
  is listed as excluded); `sensorml.PropOrganisationName` follows `ResponsibleParty.json:8` (`"organisationName"`).

### D3: Hierarchy contracts across Phase 1 catalogs

- `catalog/sosa`: `sosa.ttl` asserts zero `rdfs:subClassOf` triples; `catalog/sosa` exports constants only.
- `catalog/ssn`: `ssn.ttl:94-96` asserts two direct named parents on `sosa:Sample` (`sosa:FeatureOfInterest` and
  `sosa:Result`). `catalog/ssn` therefore exports `DirectParents(iri string) []string`, `Parents(iri string) []string`
  (breadth-first closure matching `oracle.Graph.Closure`), and `IsA(child, parent string) bool` over the integrated
  `sosa` + `ssn` + `ssn-system` closure.
- `catalog/swe` and `catalog/sensorml`: every admitted JSON Schema class has at most one `allOf` object parent, with
  seven `sensorml` classes crossing into `swe.AbstractSweIdentifiable` -> `swe.AbstractSWE`. Both packages export
  `SubClassOf(iri string) (string, bool)`, `Parents(iri string) []string`, and `IsA(child, parent string) bool`.

### D4: `swe-core` prefixes and SemEngine `v0.1.0-alpha.2` compatibility

`packs/swe-core.yaml` registers `sosa` (`http://www.w3.org/ns/sosa/`), `ssn` (`http://www.w3.org/ns/ssn/`),
`ssn-system` (`http://www.w3.org/ns/ssn/systems/`), `swe` (`http://www.opengis.net/spec/SWE/3.0/`), and `sml`
(`http://www.opengis.net/spec/sensorML/3.0/`). At SemEngine `11da4721c2233a8e2b5b1d6236a97a44e06abbb6`:

- `PrefixSpec.Validate()` (`vocabulary/pack.go:24-77`) accepts hyphenated `ssn-system`.
- `export.RegisterPrefixes` (`vocabulary/export/prefix.go:63-84`) is idempotent on `sosa` and `ssn` (`:33-34`).
- `(*prefixMap).compact` (`vocabulary/export/prefix.go:121-146`) selects longest namespace match first (`33 > 25`), so
  `ssn-system:*` compacts under `ssn-system` rather than `ssn`.

### D5: Consumer and shared-schema exclusions

`sosa:SamplingFeature` (`vocabulary/sosa/iris.go:49`) and `sosa:hasLocation` (`vocabulary/sosa/iris.go:124`) from
`semconnect` (`d0d06e00bf05a545f30ceea798db1c2b1ee47d4f`) are absent from W3C 2017 `sosa.ttl`, `ssn.ttl`, and
`ssn-system.ttl`; `sosa:hasResultingSample` and `sosa:isSamplingResultOf` are commented out in `sosa.ttl:387-403`;
`swe:nilValue` (`vocabulary/swe/iris.go:69`) is SWE 2.0 singular whereas SWE 3.0 defines `nilValues`; and RFC 5988 /
RFC 6690 Web Linking keys in `common/link.json` (`href`, `rel`, `hreflang`, `title`, `uid`, `rt`, `if`) are generic OGC
API link fields rather than SensorML 3.0 properties. All are recorded under `excluded_source_symbols` and excluded from
admission.

## Declared Cost

- SWE 3.0 and SensorML 3.0 IRIs under `http://www.opengis.net/spec/SWE/3.0/` and
  `http://www.opengis.net/spec/sensorML/3.0/` do not dereference to an OGC OWL graph unless OGC later publishes one; if
  OGC later standardizes a 3.0 OWL ontology with different IRIs, migrating consumers requires an explicit crosswalk.
- Prefixing JSON Schema property constants with `Prop` (`swe.PropUOM`, `sensorml.PropLink`) differs from OWL catalogs
  (`sosa.MadeBySensor`, `cco.HasInput`), traded deliberately so object classes and property keys never collide in Go.

## Phase 1 Acceptance Plan and Merge-Order Note

1. After owner sign-off on Phase 0, vendor `w3c/sdw` TTLs and `opengeospatial/ogcapi-connected-systems` JSON Schemas +
   licenses under `internal/oracle/testdata/`, checking every byte against the ledger `sha256` before parsing.
2. Implement `catalog/{sosa,ssn,swe,sensorml}` and `packs/swe-core` with oracle and SemEngine registration/rollback
   tests.
3. Open PR #25 bumps `@fission-ai/openspec` to `1.14.3` with zero file overlap; if PR #25 merges before Phase 0 lands,
   merge `origin/main` into this branch (`git merge origin/main`) and re-run `task verify`.
