# catalogs-and-packs Specification

## Purpose
SemVocab's ontology and schema catalogs (`bfo`, `cco`, `sosa`, `ssn`, `swe`, and `sensorml`) and its embedded packs
carry only ledger-admitted terms, with hierarchies and superclass closures taken from the pinned ontology releases and
schema bundles, and register through SemEngine's pack API.

## Requirements

### Requirement: Catalogs admit only ledger terms

`catalog/bfo`, `catalog/cco`, `catalog/sosa`, `catalog/ssn`, `catalog/swe`, and `catalog/sensorml` SHALL export
exactly the terms the admission ledger admits for them, with IRIs equal to the pinned release or schema-bundle
declaration, and SHALL NOT export any excluded source symbol.

#### Scenario: An excluded SemStreams symbol

- **WHEN** a reader looks for `PrecedesTemporally` in `catalog/bfo`
- **THEN** no such symbol exists

#### Scenario: Excluded consumer symbols in SOSA and SWE

- **WHEN** a reader looks for `SamplingFeature` or `HasLocation` in `catalog/sosa`, or `NilValue` / `PropNilValue` in
  `catalog/swe`
- **THEN** no such symbol exists

#### Scenario: Collision-free JSON Schema property constants

- **WHEN** a caller reads `swe.PropNamespace` and `sensorml.PropLink`
- **THEN** they equal `http://www.opengis.net/spec/SWE/3.0/namespace` and
  `http://www.opengis.net/spec/sensorML/3.0/link` without colliding with `swe.Namespace` or `sensorml.Link`

### Requirement: Hierarchy follows the release

For every admitted class in a catalog that declares a hierarchy, `Parents` SHALL return the transitive closure of
asserted named `rdfs:subClassOf` or JSON Schema `allOf` object parents in the pinned release, without `owl:Thing`, and
`IsA` SHALL agree with it. `catalog/sosa` SHALL export no hierarchy functions because `sosa.ttl` asserts zero
`rdfs:subClassOf` triples.

#### Scenario: A SemStreams hierarchy divergence

- **WHEN** `bfo.Parents(bfo.Site)` is called
- **THEN** it includes `BFO_0000141` (immaterial entity), as BFO 2020 asserts

#### Scenario: Multi-parent SOSA/SSN class

- **WHEN** `ssn.DirectParents(sosa.Sample)` is called
- **THEN** it returns `[]string{sosa.FeatureOfInterest, sosa.Result}`, and `ssn.IsA` is true for both

#### Scenario: Cross-catalog JSON Schema hierarchy

- **WHEN** `sensorml.Parents(sensorml.CharacteristicList)` is called
- **THEN** it returns `[]string{swe.AbstractSweIdentifiable, swe.AbstractSWE}`

### Requirement: Packs register full closures

Every entity type in an embedded pack SHALL list, in `superclasses`, the release closure of its declared parents plus
any DIEKM-local assertion recorded in the ledger and that assertion's closure. Every IRI in a pack SHALL be an
expanded `http://` or `https://` IRI within a declared prefix namespace.

#### Scenario: The DIEKM platform

- **WHEN** the `diekm` pack is loaded
- **THEN** `intel.platform` lists the 7 superclass IRIs of the Vehicle closure plus `BFO_0000030`

### Requirement: Packs register through SemEngine

The embedded packs SHALL load through SemEngine's `LoadPackYAML` and register through `RegisterPack` with
`WithPrefixRegistrar(export.RegisterPrefixes)`. A pack that fails to load or register SHALL leave registry and
serializer prefix state unchanged.

#### Scenario: An invalid entry

- **WHEN** a pack with an invalid entry is registered
- **THEN** registration fails and the registry getters and serializer prefixes return what they returned before

#### Scenario: `swe-core` compacts `ssn-system` ahead of `ssn`

- **WHEN** `packs.RegisterSWECore()` has succeeded and SemEngine exports an IRI under
  `http://www.w3.org/ns/ssn/systems/`
- **THEN** Turtle output compacts it with `ssn-system:` rather than `ssn:`
