# Catalog ownership against SemEngine

SemEngine is the source of truth for the standard constants and generic vocabulary behaviour it exports. This table
records, for SemEngine `v0.1.0-alpha.1` (`e60229334b8d8da3c0350e17802e2ea9631f91c8`), what each candidate catalog
overlaps. Dispositions and sources are in `docs/admission-ledger.yaml`.

Three kinds of evidence are kept apart:

- **Exported constant**: a public Go symbol a consumer can import.
- **Serializer prefix**: an entry in the unexported `defaultPrefixes` map (`vocabulary/export/prefix.go:24-37`). It
  compacts IRIs on output and is not a catalog API.
- **Doc mention**: an IRI in a comment or example. It is not evidence of an API.

## SemEngine files read

| File | Exported vocabulary content |
| --- | --- |
| `vocabulary/iris.go` | `SemEngineBase` (`:6`) only |
| `vocabulary/standards.go` | SKOS ×4, DC ×2 + `DCTermsTitle`, `ProvNamespace` + 3 PROV terms (rows below) |
| `vocabulary/predicates.go` | dotted predicates (`geo.location.*` `:32-36`, `graph.rel.contains` `:45`, `hierarchy.*` `:59-101`), data types (`:158-173`); no IRIs except a doc mention at `:296` |
| `vocabulary/labels.go` | `init` registers `DCTermsTitle` with `DcTitle` (`:17-21`); no new constant |
| `vocabulary/relationships.go` | `init` registers `GraphRelContains` with `ProvHadMember` (`:9-11`); no new constant |
| `vocabulary/pack.go` | `PrefixSpec`, `PredicateSpec`, `EntityTypeSpec`, `PackSpec`, `LoadPackJSON/YAML`, `RegisterPack`, `WithPrefixRegistrar`; no IRIs |
| `vocabulary/projection.go` | `ProjectionStep`, `ProjectionSpec`, `WithProjection`; no IRIs |
| `vocabulary/export/prefix.go` | `Register`, `RegisterPrefixes`; built-in prefixes rdf, owl, skos, rdfs, dc, schema, foaf, prov, ssn, sosa, xsd, ss (`:25-36`) |

`vocabulary/export/object.go:17-26` holds unexported `xsd*` and `rdfNamespace` constants, which are serializer
internals. SemEngine at this pin has no `aliases.go`, no `bfo`/`cco`/`agentic` package, and no OWL, RDF, RDFS,
Schema.org, FOAF, SOSA, SSN, GeoSPARQL, IANA, ACTR, AGNT or cyber constant.

## Per catalog

| Catalog | SemEngine exported constants | Serializer prefix | Doc mentions | SemStreams pin material | Overlap | Missing from SemEngine | Disposition |
| --- | --- | --- | --- | --- | --- | --- | --- |
| actr | none | none | none | none | none | everything (no source) | deferred |
| agnt | none | none | none | `vocabulary/agentic` (unverified mapping) | none | everything | deferred |
| bfo | none | none | none | `vocabulary/bfo/{bfo,hierarchy}.go` | none | all BFO terms | port (verified subset) |
| cco | none | none | none | `vocabulary/cco/{cco,hierarchy}.go` | none | all CCO terms | new content from CCO v2.2 (SemStreams names; 3 renamed to v2.2 labels) |
| cyber | none | none | none | none | none | everything | deferred |
| dublincore | `DcIdentifier` `standards.go:47`, `DcTitle` `:51`, `DCTermsTitle` `:60` | `dc` `prefix.go:29` | `doc.go:137` | `standards.go`: 14 constants | 3 | 11 SemStreams-only constants | use-semengine-symbol |
| foaf | none | `foaf` `prefix.go:31` | none | `FoafName`, `FoafNick`, `FoafAccountName` | none | 3 | deferred |
| geosparql | none (`geo.location.*` dotted, no IRI) | none | none | none | none | everything | deferred |
| iana | none | none | none | none | none | everything | deferred |
| owl | none | `owl` `prefix.go:26` | `predicates.go:296` | 7 `Owl*` | none | 7 | deferred |
| prov | `ProvNamespace` `:69`, `ProvWasAttributedTo` `:77`, `ProvHadMember` `:86`, `ProvGeneratedAtTime` `:94` | `prov` `prefix.go:32` | none | 48 `Prov*` | 4 | 44 | use-semengine-symbol |
| rdf | none (unexported `rdfNamespace` `object.go:26`) | `rdf` `prefix.go:25` | none | none | none | everything | deferred |
| rdfs | none | `rdfs` `prefix.go:28` | none | `RdfsLabel`, `RdfsComment`, `RdfsSeeAlso` | none | 3 | deferred |
| schema | none | `schema` = `https://schema.org/` `prefix.go:30` | `http://schema.org/batteryLevel` `doc.go:62,89,127`, `registry.go:170,300` | 7 `Schema*`, mixed http/https | none | 7 | deferred |
| skos | `SkosNotation` `:24`, `SkosBroader` `:29`, `SkosNarrower` `:34`, `SkosRelated` `:39` | `skos` `prefix.go:27` | `doc.go:136`, `predicates.go:296` | 7 `Skos*` | 4 | `SkosPrefLabel`, `SkosAltLabel`, `SkosHiddenLabel` | use-semengine-symbol |
| sosa | none | `sosa` `prefix.go:34` | none | README example only | none | everything | deferred |
| ssn | none | `ssn` `prefix.go:33` | none | README example only | none | everything | deferred |

## Divergences to keep explicit

- Schema.org: SemEngine's serializer binds `https://schema.org/`, while SemEngine's own doc examples and three
  SemStreams constants use `http://schema.org/`. Any future schema catalog needs a namespace ruling and a mapping.
- BFO and CCO: SemVocab admits only terms declared in BFO 2020 (`release-2024-01-29`) and CCO `v2.2`. Its hierarchy
  follows those releases, not the SemStreams map, where they differ. CCO v2 uses opaque IRIs under
  `https://www.commoncoreontologies.org/`, so no SemStreams CCO IRI is carried (`docs/admission-ledger.yaml`).
- No SemVocab package copies an IRI string that SemEngine exports. The three use-semengine-symbol catalogs have no
  SemVocab API.
