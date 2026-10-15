# catalogs-and-packs Specification

## Purpose
SemVocab's `bfo` and `cco` catalogs and its embedded packs carry only ledger-admitted terms, with hierarchies and
superclass closures taken from the pinned ontology releases, and register through SemEngine's pack API.

## Requirements

### Requirement: Catalogs admit only ledger terms

`catalog/bfo` and `catalog/cco` SHALL export exactly the terms the admission ledger admits for them, with IRIs equal
to the release declaration, and SHALL NOT export any excluded source symbol.

#### Scenario: An excluded SemStreams symbol

- **WHEN** a reader looks for `PrecedesTemporally` in `catalog/bfo`
- **THEN** no such symbol exists

### Requirement: Hierarchy follows the release

For every admitted class, `Parents` SHALL return the transitive closure of asserted named `rdfs:subClassOf` in the
pinned release, without `owl:Thing`, and `IsA` SHALL agree with it.

#### Scenario: A SemStreams hierarchy divergence

- **WHEN** `bfo.Parents(bfo.Site)` is called
- **THEN** it includes `BFO_0000141` (immaterial entity), as BFO 2020 asserts

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
