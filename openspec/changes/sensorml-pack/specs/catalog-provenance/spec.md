# catalog-provenance

## MODIFIED Requirements

### Requirement: One disposition per candidate

The ledger SHALL record one entry for each of 19 catalogs (`actr`, `agnt`, `bfo`, `cco`, `cyber`, `dublincore`, `foaf`,
`geosparql`, `iana`, `owl`, `prov`, `rdf`, `rdfs`, `schema`, `skos`, `sosa`, `ssn`, `swe`, `sensorml`) and 4 packs
(`diekm`, `bfo-cco-core`, `swe-core`, `sensorml`), disambiguated by kind when names match. Each entry SHALL carry one
disposition from `use-semengine-symbol`, `thin-reexport`, `new-content`, `port`, `extraction`, `deferred`.

#### Scenario: A catalog with no source

- **WHEN** a reader looks up `actr`
- **THEN** its disposition is `deferred` and its reason says no source exists at the SemStreams pin

#### Scenario: OGC 3.0 and W3C SOSA/SSN candidates

- **WHEN** a reader looks up catalogs `sosa`, `ssn`, `swe`, `sensorml` and pack `swe-core`
- **THEN** each entry carries disposition `new-content` for Phase 1

#### Scenario: Disambiguating catalog and pack sensorml

- **WHEN** a caller looks up `sensorml` in `internal/ledger`
- **THEN** `Catalog("sensorml")` returns the Phase 1 catalog entry, `Pack("sensorml")` returns the Phase 2 pack entry
  with disposition `new-content`, and untyped `Entry("sensorml")` fails closed naming `Catalog` or `Pack`

## ADDED Requirements

### Requirement: Explicit cross-standard local assertions

When a pack aligns classes across pinned standards that do not assert `rdfs:subClassOf` triples between each other, its
ledger entry SHALL record each `<class> rdfs:subClassOf <parent> (<rationale>)` alignment under `local_assertions`, and
tests SHALL verify `superclasses` against the union of the pinned catalog hierarchies and `local_assertions`.

#### Scenario: SensorML 3.0 to W3C SOSA/SSN alignments on pack sensorml

- **WHEN** a reader inspects `local_assertions` on `pack: sensorml`
- **THEN** it records four `rdfs:subClassOf` alignments (`sml:PhysicalSystem -> ssn:System`,
  `sml:SimpleProcess -> sosa:Procedure`, `sml:Deployment -> ssn:Deployment`, and
  `sml:DerivedProperty -> sosa:ObservableProperty`) with rationales citing ruling R5 on #26
